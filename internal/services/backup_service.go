package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// BackupInfo describes one backup file.
type BackupInfo struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

const backupPrefix, backupSuffix = "exams-", ".db"

// BackupService writes consistent SQLite snapshots, keeping the newest Keep
// files in Dir.
type BackupService struct {
	repo models.ExamRepository
	dir  string
	keep int
	now  func() time.Time
}

// NewBackupService creates a BackupService. dir may be empty, in which case
// only on-demand downloads (WriteTo) are available.
func NewBackupService(repo models.ExamRepository, dir string, keep int) *BackupService {
	if keep < 1 {
		keep = 1
	}
	return &BackupService{repo: repo, dir: dir, keep: keep, now: time.Now}
}

// Enabled reports whether scheduled backups to a directory are configured.
func (s *BackupService) Enabled() bool {
	return s.dir != ""
}

// Snapshot writes a new backup into the backup directory and prunes old ones.
func (s *BackupService) Snapshot(ctx context.Context) (BackupInfo, error) {
	if !s.Enabled() {
		return BackupInfo{}, fmt.Errorf("backups are not configured (set BACKUP_DIR)")
	}
	if err := os.MkdirAll(s.dir, 0750); err != nil {
		return BackupInfo{}, fmt.Errorf("failed to create backup directory: %w", err)
	}

	name := backupPrefix + s.now().UTC().Format("20060102-150405.000") + backupSuffix
	path := filepath.Join(s.dir, name)
	if err := s.repo.Backup(ctx, path); err != nil {
		return BackupInfo{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return BackupInfo{}, err
	}
	if err := s.prune(); err != nil {
		log.Printf("⚠️ Could not prune old backups: %v", err)
	}
	log.Printf("💾 Backup written: %s (%d bytes)", path, info.Size())
	return BackupInfo{Name: name, Size: info.Size(), CreatedAt: info.ModTime()}, nil
}

// List returns the backups in the backup directory, newest first.
func (s *BackupService) List() ([]BackupInfo, error) {
	out := []BackupInfo{}
	if !s.Enabled() {
		return out, nil
	}
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, backupPrefix) || !strings.HasSuffix(name, backupSuffix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupInfo{Name: name, Size: info.Size(), CreatedAt: info.ModTime()})
	}
	// Names embed a sortable UTC timestamp.
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out, nil
}

func (s *BackupService) prune() error {
	backups, err := s.List()
	if err != nil {
		return err
	}
	for _, b := range backups[min(len(backups), s.keep):] {
		if err := os.Remove(filepath.Join(s.dir, b.Name)); err != nil {
			return err
		}
	}
	return nil
}

// WriteTo streams a fresh snapshot to w (for downloads).
func (s *BackupService) WriteTo(ctx context.Context, w io.Writer) error {
	tmpDir, err := os.MkdirTemp("", "exams-backup-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	path := filepath.Join(tmpDir, "exams.db")
	if err := s.repo.Backup(ctx, path); err != nil {
		return err
	}
	f, err := os.Open(path) // #nosec G304 -- temp file created above
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// Schedule takes a snapshot every interval until ctx is cancelled.
func (s *BackupService) Schedule(ctx context.Context, interval time.Duration) {
	if !s.Enabled() {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.Snapshot(ctx); err != nil {
				log.Printf("⚠️ Scheduled backup failed: %v", err)
			}
		}
	}
}

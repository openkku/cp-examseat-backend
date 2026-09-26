package services

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

type backupRepo struct {
	models.ExamRepository
	calls int
}

func (b *backupRepo) Backup(ctx context.Context, dest string) error {
	b.calls++
	return os.WriteFile(dest, []byte("SQLite format 3\x00"), 0600)
}

func TestBackupSnapshotAndRetention(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "backups")
	repo := &backupRepo{}
	svc := NewBackupService(repo, dir, 3)
	clock := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { clock = clock.Add(time.Hour); return clock }

	for i := 0; i < 5; i++ {
		if _, err := svc.Snapshot(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	// Unrelated files are never touched.
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep"), 0600)

	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("kept %d backups; want 3: %+v", len(list), list)
	}
	if list[0].Name != "exams-20260926-170000.000.db" || list[2].Name != "exams-20260926-150000.000.db" {
		t.Errorf("unexpected retention order: %+v", list)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
		t.Errorf("prune removed an unrelated file")
	}
}

func TestBackupWriteToAndDisabled(t *testing.T) {
	svc := NewBackupService(&backupRepo{}, "", 7)
	if svc.Enabled() {
		t.Fatal("empty dir must disable scheduled backups")
	}
	if _, err := svc.Snapshot(context.Background()); err == nil {
		t.Error("Snapshot without BACKUP_DIR must fail")
	}
	if list, _ := svc.List(); list == nil || len(list) != 0 {
		t.Errorf("List without dir = %#v; want empty slice", list)
	}

	var buf bytes.Buffer
	if err := svc.WriteTo(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("SQLite format 3")) {
		t.Errorf("download content = %q", buf.String())
	}
}

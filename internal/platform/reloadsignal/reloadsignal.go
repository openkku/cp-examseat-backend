// Package reloadsignal lets tools that change data (the operator app, the
// migrate CLI) tell a running server to reload, through a marker file in the
// shared data directory. It works across processes and containers, with no
// network port and no signal permissions.
package reloadsignal

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"
)

// FileName is the marker file inside the data directory.
const FileName = ".reload"

// Path returns the marker file of dataDir.
func Path(dataDir string) string {
	return filepath.Join(dataDir, FileName)
}

// Touch asks a server using dataDir to reload by updating the marker's mtime.
func Touch(dataDir string) error {
	path := Path(dataDir)
	now := time.Now()
	if err := os.Chtimes(path, now, now); err == nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 -- fixed name inside DATA_DIR
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Chtimes(path, now, now)
}

// Notifier signals reloads for one data directory. It implements the Reloader
// and Publisher ports of the admin and roomconfig slices.
type Notifier struct {
	DataDir string
}

// Reload asks the server to reload everything.
func (n Notifier) Reload(context.Context) error {
	return Touch(n.DataDir)
}

// RoomsChanged asks the server to reload after room files changed.
func (n Notifier) RoomsChanged() error {
	return Touch(n.DataDir)
}

// Watch calls reload whenever the marker of dataDir changes, checking every
// interval until ctx is done. A marker that already exists at start counts as
// seen.
func Watch(ctx context.Context, dataDir string, interval time.Duration, reload func(context.Context) error) {
	path := Path(dataDir)
	last := modTime(path)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if mt := modTime(path); !mt.Equal(last) {
				last = mt
				log.Printf("🔔 %s changed, reloading", FileName)
				if err := reload(ctx); err != nil {
					log.Printf("⚠️ Reload failed: %v", err)
				}
			}
		}
	}
}

func modTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

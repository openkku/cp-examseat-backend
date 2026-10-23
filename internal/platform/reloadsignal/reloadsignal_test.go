package reloadsignal

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatchReloadsOnTouch(t *testing.T) {
	dir := t.TempDir()
	if err := Touch(dir); err != nil { // existing marker: counts as seen
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var reloads atomic.Int32
	done := make(chan struct{})
	go func() {
		Watch(ctx, dir, 10*time.Millisecond, func(context.Context) error { reloads.Add(1); return nil })
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	if n := reloads.Load(); n != 0 {
		t.Fatalf("reloaded %d times before any touch", n)
	}

	// Some filesystems keep mtimes at 1s resolution; make the change visible.
	later := time.Now().Add(2 * time.Second)
	if err := Touch(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(Path(dir), later, later); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for reloads.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := reloads.Load(); n != 1 {
		t.Fatalf("reloads = %d, want 1", n)
	}

	cancel()
	<-done
}

func TestNotifierTouchesMarker(t *testing.T) {
	dir := t.TempDir()
	n := Notifier{DataDir: dir}
	if err := n.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if modTime(Path(dir)).IsZero() {
		t.Fatal("marker not created")
	}
	if err := n.RoomsChanged(); err != nil {
		t.Fatal(err)
	}
}

package roomconfig_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/room"
	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
	"github.com/openkku/cp-examseat-backend/internal/roomconfig/filesystem"
)

// countingPublisher counts the reloads a Service asks for.
type countingPublisher struct {
	calls int
	err   error
}

func (p *countingPublisher) RoomsChanged() error { p.calls++; return p.err }

func TestServiceLifecycle(t *testing.T) {
	dir := t.TempDir()
	pub := &countingPublisher{}
	store := roomconfig.NewService(filesystem.NewStore(dir), pub)

	raw, err := store.RawMetadata()
	if err != nil || string(raw) != "{}" {
		t.Fatalf("RawMetadata on empty dir = %q, %v", raw, err)
	}
	if rooms, err := store.Rooms(); err != nil || len(rooms) != 0 {
		t.Fatalf("Rooms on empty dir = %v, %v", rooms, err)
	}
	if err := store.DeleteRoom("X"); !errors.Is(err, roomconfig.ErrMetadataNotFound) {
		t.Errorf("DeleteRoom without metadata = %v; want ErrMetadataNotFound", err)
	}

	meta := room.Meta{LayoutFile: "shared.json", LayoutImage: "/room/image/a.jpg"}
	if err := store.SaveRoom("A", meta); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRoom("B", meta); err != nil {
		t.Fatal(err)
	}

	if rooms, err := store.Rooms(); err != nil || len(rooms) != 2 || rooms["A"].LayoutFile != "shared.json" {
		t.Errorf("Rooms = %v, %v", rooms, err)
	}

	layout, err := store.Layout("shared.json")
	if err != nil || string(layout) != `{"layout": []}` {
		t.Errorf("new room should get an empty layout file, got %q, %v", layout, err)
	}

	if pub.calls != 2 {
		t.Errorf("reloads after two saves = %d; want 2", pub.calls)
	}

	if _, err := store.SaveLayout("shared.json", []byte(`[1,2]`)); !errors.Is(err, roomconfig.ErrInvalidLayout) {
		t.Errorf("SaveLayout with non-object = %v; want ErrInvalidLayout", err)
	}
	if _, err := store.SaveLayout("../../escape.json", []byte(`{"layout": []}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "map", "escape.json")); err != nil {
		t.Errorf("layout names must be confined to map/: %v", err)
	}

	if err := store.DeleteRoom("missing"); !errors.Is(err, roomconfig.ErrRoomNotFound) {
		t.Errorf("DeleteRoom(missing) = %v; want ErrRoomNotFound", err)
	}

	// The layout file survives while another room still uses it.
	if err := store.DeleteRoom("A"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "map", "shared.json")); err != nil {
		t.Errorf("shared layout removed while still in use: %v", err)
	}
	if err := store.DeleteRoom("B"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "map", "shared.json")); !os.IsNotExist(err) {
		t.Errorf("unused layout should be removed, stat err = %v", err)
	}

	raw, _ = store.RawMetadata()
	if strings.TrimSpace(string(raw)) != "{}" {
		t.Errorf("metadata after deleting all rooms = %s", raw)
	}
	// 2 rooms saved, 1 valid layout saved, 2 rooms deleted; failures publish nothing.
	if pub.calls != 5 {
		t.Errorf("reloads = %d; want 5", pub.calls)
	}
}

func TestServiceReportsFailedReloadSignal(t *testing.T) {
	pub := &countingPublisher{err: errors.New("read-only volume")}
	svc := roomconfig.NewService(filesystem.NewStore(t.TempDir()), pub)
	err := svc.SaveRoom("A", room.Meta{LayoutFile: "a.json"})
	if err == nil || !strings.Contains(err.Error(), "saved, but") {
		t.Errorf("SaveRoom with failing publisher = %v; want a saved-but error", err)
	}
}

func TestRawLayoutReportsMissingFiles(t *testing.T) {
	svc := roomconfig.NewService(filesystem.NewStore(t.TempDir()), &countingPublisher{})
	if _, exists, err := svc.RawLayout("none.json"); err != nil || exists {
		t.Fatalf("RawLayout(missing) = exists %v, %v; want false, nil", exists, err)
	}
	if _, err := svc.SaveLayout("a.json", []byte(`{"layout":[]}`)); err != nil {
		t.Fatal(err)
	}
	if data, exists, err := svc.RawLayout("a.json"); err != nil || !exists || !strings.Contains(string(data), "layout") {
		t.Errorf("RawLayout(a.json) = %q, %v, %v", data, exists, err)
	}
}

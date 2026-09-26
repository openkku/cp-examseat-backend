package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

func TestRoomConfigStoreLifecycle(t *testing.T) {
	dir := t.TempDir()
	store := NewRoomConfigStore(dir)

	raw, err := store.RawMetadata()
	if err != nil || string(raw) != "{}" {
		t.Fatalf("RawMetadata on empty dir = %q, %v", raw, err)
	}
	if err := store.DeleteRoom("X"); !errors.Is(err, ErrMetadataNotFound) {
		t.Errorf("DeleteRoom without metadata = %v; want ErrMetadataNotFound", err)
	}

	meta := models.RoomMeta{LayoutFile: "shared.json", LayoutImage: "/room/image/a.jpg"}
	if err := store.SaveRoom("A", meta); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRoom("B", meta); err != nil {
		t.Fatal(err)
	}

	layout, err := store.Layout("shared.json")
	if err != nil || string(layout) != `{"layout": []}` {
		t.Errorf("new room should get an empty layout file, got %q, %v", layout, err)
	}

	if err := store.SaveLayout("shared.json", []byte(`[1,2]`)); !errors.Is(err, ErrInvalidLayout) {
		t.Errorf("SaveLayout with non-object = %v; want ErrInvalidLayout", err)
	}
	if err := store.SaveLayout("../../escape.json", []byte(`{"layout": [1]}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "map", "escape.json")); err != nil {
		t.Errorf("layout names must be confined to map/: %v", err)
	}

	if err := store.DeleteRoom("missing"); !errors.Is(err, ErrRoomNotFound) {
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
}

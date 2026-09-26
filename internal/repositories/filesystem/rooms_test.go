package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRoomCatalog(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "metadata.json"), `{
		"CP.9127": {"layout_file": "CP9127.json", "layout_image": "/room/image/CP.9127.jpg", "map_url": "https://maps.test", "images": ["/room/image/a.jpg", "https://other.test/b.jpg"]},
		"SC.1101": {"layout_file": "SC1101.json", "layout_image": "", "map_url": "", "images": null},
		"BROKEN":  {"layout_file": "missing.json"}
	}`)
	writeFile(t, filepath.Join(dir, "map", "CP9127.json"), `{"frontLabel": "Board", "layout": [{"type": "aisle", "width": 1}]}`)
	writeFile(t, filepath.Join(dir, "map", "SC1101.json"), `{"layout": []}`)

	catalog := LoadRoomCatalog(dir, "https://cdn.test/")

	if catalog.Configured != 3 {
		t.Errorf("Configured = %d; want 3", catalog.Configured)
	}
	if len(catalog.Rooms) != 2 {
		t.Fatalf("loaded rooms = %d; want 2 (BROKEN has no layout file)", len(catalog.Rooms))
	}

	cp := catalog.Rooms["CP.9127"]
	if cp.LayoutImage != "https://cdn.test/room/image/CP.9127.jpg" {
		t.Errorf("layout image = %q; want CDN-prefixed", cp.LayoutImage)
	}
	if cp.Images[0] != "https://cdn.test/room/image/a.jpg" || cp.Images[1] != "https://other.test/b.jpg" {
		t.Errorf("images = %v; only root-relative paths get the CDN prefix", cp.Images)
	}
	if cp.Labels["frontLabel"] != "Board" {
		t.Errorf("labels = %v", cp.Labels)
	}
	if _, ok := cp.Labels["backLabel"]; ok {
		t.Errorf("absent backLabel must not be set")
	}

	sc := catalog.Rooms["SC.1101"]
	if sc.Images == nil || len(sc.Images) != 0 {
		t.Errorf("images of a room without photos should be an empty slice, got %#v", sc.Images)
	}
}

func TestLoadRoomCatalogMissingMetadata(t *testing.T) {
	catalog := LoadRoomCatalog(t.TempDir(), "")
	if catalog.Configured != 0 || len(catalog.Rooms) != 0 || catalog.Rooms == nil {
		t.Errorf("expected empty usable catalog, got %+v", catalog)
	}
}

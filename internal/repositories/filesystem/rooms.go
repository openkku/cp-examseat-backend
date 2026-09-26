// Package filesystem persists room metadata and seating layouts as JSON files
// under $DATA_DIR/room (metadata.json, map/*.json, image/*).
package filesystem

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// LoadRoomCatalog reads metadata.json and every referenced layout file from
// roomDir. Rooms whose layout is missing or invalid are logged and skipped; a
// missing or invalid metadata.json yields an empty catalog.
//
// When imageBaseURL is set, root-relative image paths are rewritten onto it.
func LoadRoomCatalog(roomDir string, imageBaseURL string) models.RoomCatalog {
	catalog := models.RoomCatalog{Rooms: make(map[string]models.Room)}

	metaPath := filepath.Join(roomDir, "metadata.json")
	metaContent, err := os.ReadFile(metaPath) // #nosec G304 -- path from DATA_DIR configuration
	if err != nil {
		log.Printf("⚠️ Warning: Could not read room metadata at %s: %v. Starting with empty layouts.", metaPath, err)
		return catalog
	}

	var metadata map[string]models.RoomMeta
	if err := json.Unmarshal(metaContent, &metadata); err != nil {
		log.Printf("❌ Error: Could not parse room metadata at %s: %v. Starting with empty layouts.", metaPath, err)
		return catalog
	}
	catalog.Configured = len(metadata)

	imageBaseURL = strings.TrimSuffix(imageBaseURL, "/")
	prefixURL := func(urlPath string) string {
		if imageBaseURL != "" && strings.HasPrefix(urlPath, "/") {
			return imageBaseURL + "/" + strings.TrimPrefix(urlPath, "/")
		}
		return urlPath
	}

	mapDir := filepath.Join(roomDir, "map")
	for name, meta := range metadata {
		// Base confines layout files to map/, like the room-config tool does.
		content, err := os.ReadFile(filepath.Join(mapDir, filepath.Base(meta.LayoutFile))) // #nosec G304 -- confined to map/
		if err != nil {
			log.Printf("⚠️  Warning: Room '%s' mapped to '%s' but file is missing/unreadable.", name, meta.LayoutFile)
			continue
		}

		var layoutFile map[string]any
		if err := json.Unmarshal(content, &layoutFile); err != nil {
			log.Printf("❌ Error parsing JSON for %s: %v", name, err)
			continue
		}

		images := make([]string, 0, len(meta.Images))
		for _, img := range meta.Images {
			images = append(images, prefixURL(img))
		}

		labels := make(map[string]any)
		for _, key := range []string{"frontLabel", "backLabel"} {
			if v, ok := layoutFile[key]; ok {
				labels[key] = v
			}
		}

		catalog.Rooms[name] = models.Room{
			Name:        name,
			LayoutImage: prefixURL(meta.LayoutImage),
			MapURL:      meta.MapURL,
			Images:      images,
			Layout:      layoutFile["layout"],
			Labels:      labels,
		}
	}

	log.Printf("✅ Loaded %d room layouts into memory.", len(catalog.Rooms))
	return catalog
}

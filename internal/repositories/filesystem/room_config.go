package filesystem

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// ErrRoomNotFound is returned when deleting a room absent from metadata.json.
var ErrRoomNotFound = errors.New("room not found")

// ErrMetadataNotFound is returned when metadata.json does not exist yet.
var ErrMetadataNotFound = errors.New("metadata config file not found")

// emptyLayout is written for new rooms and returned for missing layout files.
var emptyLayout = []byte(`{"layout": []}`)

// RoomConfigStore edits metadata.json and the layout files used by the
// room-config tool.
type RoomConfigStore struct {
	roomDir string
}

// NewRoomConfigStore manages the room configuration stored in roomDir.
func NewRoomConfigStore(roomDir string) *RoomConfigStore {
	return &RoomConfigStore{roomDir: roomDir}
}

func (s *RoomConfigStore) metadataPath() string {
	os.MkdirAll(s.roomDir, 0755)
	return filepath.Join(s.roomDir, "metadata.json")
}

func (s *RoomConfigStore) mapDir() string {
	dir := filepath.Join(s.roomDir, "map")
	os.MkdirAll(dir, 0755)
	return dir
}

// layoutPath resolves a layout file name inside map/, dropping any directory
// components to prevent path traversal.
func (s *RoomConfigStore) layoutPath(filename string) string {
	return filepath.Join(s.mapDir(), filepath.Base(filename))
}

// RawMetadata returns metadata.json as stored, or "{}" if it does not exist.
func (s *RoomConfigStore) RawMetadata() ([]byte, error) {
	data, err := os.ReadFile(s.metadataPath())
	if os.IsNotExist(err) {
		return []byte("{}"), nil
	}
	return data, err
}

func (s *RoomConfigStore) readMetadata() (map[string]models.RoomMeta, error) {
	config := make(map[string]models.RoomMeta)
	data, err := os.ReadFile(s.metadataPath())
	if err != nil {
		return config, err
	}
	json.Unmarshal(data, &config)
	return config, nil
}

func (s *RoomConfigStore) writeMetadata(config map[string]models.RoomMeta) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metadataPath(), data, 0644)
}

// SaveRoom creates or replaces a room entry, creating an empty layout file
// for it when the referenced file does not exist yet.
func (s *RoomConfigStore) SaveRoom(roomID string, meta models.RoomMeta) error {
	config, _ := s.readMetadata()
	config[roomID] = meta

	if meta.LayoutFile != "" {
		layoutPath := s.layoutPath(meta.LayoutFile)
		if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
			os.WriteFile(layoutPath, emptyLayout, 0644)
		}
	}

	return s.writeMetadata(config)
}

// DeleteRoom removes a room entry and its layout file when no other room uses it.
func (s *RoomConfigStore) DeleteRoom(roomID string) error {
	config, err := s.readMetadata()
	if err != nil {
		return ErrMetadataNotFound
	}

	meta, exists := config[roomID]
	if !exists {
		return ErrRoomNotFound
	}
	delete(config, roomID)

	if err := s.writeMetadata(config); err != nil {
		return err
	}

	if meta.LayoutFile != "" {
		for _, other := range config {
			if other.LayoutFile == meta.LayoutFile {
				return nil
			}
		}
		os.Remove(s.layoutPath(meta.LayoutFile))
	}
	return nil
}

// Layout returns a layout file, or an empty layout if it does not exist.
func (s *RoomConfigStore) Layout(filename string) ([]byte, error) {
	data, err := os.ReadFile(s.layoutPath(filename))
	if os.IsNotExist(err) {
		return emptyLayout, nil
	}
	return data, err
}

// SaveLayout validates that body is a JSON object and writes it as a layout file.
func (s *RoomConfigStore) SaveLayout(filename string, body []byte) error {
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidLayout, err)
	}
	return os.WriteFile(s.layoutPath(filename), body, 0644)
}

// ErrInvalidLayout is returned by SaveLayout for a body that is not a JSON object.
var ErrInvalidLayout = errors.New("invalid layout JSON")

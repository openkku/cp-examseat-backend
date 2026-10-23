// Package filesystem stores the room configuration as JSON files under
// $DATA_DIR/room (metadata.json, map/*.json).
package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/openkku/cp-examseat-backend/internal/room"
	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
)

// Store implements roomconfig.Store in a room directory.
type Store struct {
	roomDir string
}

var _ roomconfig.Store = (*Store)(nil)

// NewStore manages the room configuration stored in roomDir.
func NewStore(roomDir string) *Store {
	return &Store{roomDir: roomDir}
}

func (s *Store) metadataPath() string {
	os.MkdirAll(s.roomDir, 0755) // #nosec G301 -- room assets are public, served to every visitor
	return filepath.Join(s.roomDir, "metadata.json")
}

func (s *Store) mapDir() string {
	dir := filepath.Join(s.roomDir, "map")
	os.MkdirAll(dir, 0755) // #nosec G301 -- room assets are public, served to every visitor
	return dir
}

// layoutPath resolves a layout file name inside map/, dropping any directory
// components to prevent path traversal.
func (s *Store) layoutPath(filename string) string {
	return filepath.Join(s.mapDir(), filepath.Base(filename))
}

// RawMetadata returns metadata.json as stored.
func (s *Store) RawMetadata() ([]byte, bool, error) {
	return readFile(s.metadataPath())
}

// ReadMetadata parses metadata.json; an unparsable file reads as empty.
func (s *Store) ReadMetadata() (map[string]room.Meta, error) {
	config := make(map[string]room.Meta)
	data, err := os.ReadFile(s.metadataPath())
	if err != nil {
		return config, err
	}
	json.Unmarshal(data, &config)
	return config, nil
}

// WriteMetadata replaces metadata.json.
func (s *Store) WriteMetadata(config map[string]room.Meta) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metadataPath(), data, 0644) // #nosec G306 -- public room metadata
}

// ReadLayout returns a layout file from map/.
func (s *Store) ReadLayout(filename string) ([]byte, bool, error) {
	return readFile(s.layoutPath(filename))
}

// WriteLayout writes a layout file into map/.
func (s *Store) WriteLayout(filename string, data []byte) error {
	return os.WriteFile(s.layoutPath(filename), data, 0644) // #nosec G306 -- public room layout
}

// RemoveLayout deletes a layout file from map/.
func (s *Store) RemoveLayout(filename string) error {
	return os.Remove(s.layoutPath(filename))
}

func readFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- confined to the room directory
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return data, err == nil, err
}

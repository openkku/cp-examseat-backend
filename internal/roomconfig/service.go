// Package roomconfig backs the room-config tool: editing the room metadata
// (room/metadata.json) and the seating layout files (room/map/*.json).
package roomconfig

import (
	"encoding/json"
	"fmt"

	"github.com/openkku/cp-examseat-backend/internal/room"
)

// emptyLayout is written for new rooms and returned for missing layout files.
var emptyLayout = []byte(`{"layout": []}`)

// Service implements the room-config use cases.
type Service struct {
	store Store
}

// NewService creates a Service on store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// RawMetadata returns the metadata as stored, or "{}" if it does not exist.
func (s *Service) RawMetadata() ([]byte, error) {
	data, exists, err := s.store.RawMetadata()
	if err == nil && !exists {
		return []byte("{}"), nil
	}
	return data, err
}

// SaveRoom creates or replaces a room entry, creating an empty layout file
// for it when the referenced file does not exist yet.
func (s *Service) SaveRoom(roomID string, meta room.Meta) error {
	config, err := s.store.ReadMetadata()
	if err != nil || config == nil {
		config = make(map[string]room.Meta)
	}
	config[roomID] = meta

	if meta.LayoutFile != "" {
		if _, exists, err := s.store.ReadLayout(meta.LayoutFile); err == nil && !exists {
			s.store.WriteLayout(meta.LayoutFile, emptyLayout)
		}
	}

	return s.store.WriteMetadata(config)
}

// DeleteRoom removes a room entry and its layout file when no other room uses it.
func (s *Service) DeleteRoom(roomID string) error {
	config, err := s.store.ReadMetadata()
	if err != nil {
		return ErrMetadataNotFound
	}

	meta, exists := config[roomID]
	if !exists {
		return ErrRoomNotFound
	}
	delete(config, roomID)

	if err := s.store.WriteMetadata(config); err != nil {
		return err
	}

	if meta.LayoutFile != "" {
		for _, other := range config {
			if other.LayoutFile == meta.LayoutFile {
				return nil
			}
		}
		s.store.RemoveLayout(meta.LayoutFile)
	}
	return nil
}

// Layout returns a layout file, or an empty layout if it does not exist.
func (s *Service) Layout(filename string) ([]byte, error) {
	data, exists, err := s.store.ReadLayout(filename)
	if err == nil && !exists {
		return emptyLayout, nil
	}
	return data, err
}

// SaveLayout validates that body is a JSON object and writes it as a layout file.
func (s *Service) SaveLayout(filename string, body []byte) error {
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidLayout, err)
	}
	return s.store.WriteLayout(filename, body)
}

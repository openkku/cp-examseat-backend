// Package roomconfig backs the room-config tool: editing the room metadata
// (room/metadata.json) and the seating layout files (room/map/*.json).
package roomconfig

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openkku/cp-examseat-backend/internal/room"
)

// ErrRoomNotFound is returned when deleting a room absent from the metadata.
var ErrRoomNotFound = errors.New("room not found")

// ErrMetadataNotFound is returned when the metadata does not exist yet.
var ErrMetadataNotFound = errors.New("metadata config file not found")

// ErrInvalidLayout is returned by SaveLayout for a body that is not a JSON object.
var ErrInvalidLayout = errors.New("invalid layout JSON")

// emptyLayout is written for new rooms and returned for missing layout files.
var emptyLayout = []byte(`{"layout": []}`)

// Store reads and writes the stored room configuration. Layout file names may
// contain directory components; the store confines them to its layout folder.
type Store interface {
	// RawMetadata returns the metadata as stored; exists is false when there is none.
	RawMetadata() (data []byte, exists bool, err error)
	// ReadMetadata returns the metadata entries; it fails when there is none.
	ReadMetadata() (map[string]room.Meta, error)
	WriteMetadata(config map[string]room.Meta) error
	// ReadLayout returns a layout file; exists is false when there is none.
	ReadLayout(filename string) (data []byte, exists bool, err error)
	WriteLayout(filename string, data []byte) error
	RemoveLayout(filename string) error
}

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

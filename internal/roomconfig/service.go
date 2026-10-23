// Package roomconfig backs room editing in the operator app: the room
// metadata (room/metadata.json) and the seating layout files (room/map/*.json).
// Every change is published so the running server reloads its rooms.
package roomconfig

import (
	"fmt"

	"github.com/openkku/cp-examseat-backend/internal/room"
)

// emptyLayout is written for new rooms and returned for missing layout files.
var emptyLayout = []byte(`{"layout": []}`)

// Service implements the room editing use cases.
type Service struct {
	store     Store
	publisher Publisher
}

// NewService creates a Service on store that announces changes to publisher.
func NewService(store Store, publisher Publisher) *Service {
	return &Service{store: store, publisher: publisher}
}

// published announces a successful change; err is the change's own result.
func (s *Service) published(err error) error {
	if err != nil {
		return err
	}
	if err := s.publisher.RoomsChanged(); err != nil {
		return fmt.Errorf("saved, but asking the server to reload failed: %w", err)
	}
	return nil
}

// RawMetadata returns the metadata as stored, or "{}" if it does not exist.
func (s *Service) RawMetadata() ([]byte, error) {
	data, exists, err := s.store.RawMetadata()
	if err == nil && !exists {
		return []byte("{}"), nil
	}
	return data, err
}

// Rooms returns the room entries, or none if the metadata does not exist yet.
func (s *Service) Rooms() (map[string]room.Meta, error) {
	_, exists, err := s.store.RawMetadata()
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]room.Meta{}, nil
	}
	return s.store.ReadMetadata()
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

	return s.published(s.store.WriteMetadata(config))
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

	if meta.LayoutFile != "" && !layoutInUse(config, meta.LayoutFile) {
		s.store.RemoveLayout(meta.LayoutFile)
	}
	return s.published(nil)
}

func layoutInUse(config map[string]room.Meta, layoutFile string) bool {
	for _, other := range config {
		if other.LayoutFile == layoutFile {
			return true
		}
	}
	return false
}

// Layout returns a layout file, or an empty layout if it does not exist.
func (s *Service) Layout(filename string) ([]byte, error) {
	data, exists, err := s.store.ReadLayout(filename)
	if err == nil && !exists {
		return emptyLayout, nil
	}
	return data, err
}

// RawLayout returns a layout file as stored; exists is false when there is none.
func (s *Service) RawLayout(filename string) (data []byte, exists bool, err error) {
	return s.store.ReadLayout(filename)
}

// SaveLayout validates that body is a valid layout JSON, writes it as a layout
// file, and returns the number of seats.
func (s *Service) SaveLayout(filename string, body []byte) (int, error) {
	layout, err := ParseLayout(body)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidLayout, err)
	}
	seatCount := len(layout.SeatIDs())
	if err := s.published(s.store.WriteLayout(filename, body)); err != nil {
		return 0, err
	}
	return seatCount, nil
}

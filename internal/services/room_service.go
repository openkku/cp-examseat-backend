package services

import "github.com/openkku/cp-examseat-backend/internal/models"

// RoomService serves the room catalog (layouts, images, map links) loaded at startup.
type RoomService struct {
	catalog models.RoomCatalog
}

// NewRoomService wraps a loaded room catalog.
func NewRoomService(catalog models.RoomCatalog) *RoomService {
	if catalog.Rooms == nil {
		catalog.Rooms = make(map[string]models.Room)
	}
	return &RoomService{catalog: catalog}
}

// All returns every room with a loaded layout, keyed by name.
func (s *RoomService) All() map[string]models.Room {
	return s.catalog.Rooms
}

// Find returns the rooms among names that exist, keyed by name.
func (s *RoomService) Find(names []string) map[string]models.Room {
	found := make(map[string]models.Room)
	for _, name := range names {
		if room, ok := s.catalog.Rooms[name]; ok {
			found[name] = room
		}
	}
	return found
}

// Has reports whether a room has a loaded layout.
func (s *RoomService) Has(name string) bool {
	_, ok := s.catalog.Rooms[name]
	return ok
}

// Count is the number of rooms with a loaded layout.
func (s *RoomService) Count() int {
	return len(s.catalog.Rooms)
}

// ConfiguredCount is the number of rooms listed in metadata.json.
func (s *RoomService) ConfiguredCount() int {
	return s.catalog.Configured
}

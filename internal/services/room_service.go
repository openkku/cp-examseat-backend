package services

import (
	"sync"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// RoomService serves the room catalog (layouts, images, map links). The
// catalog can be swapped at runtime when room files change.
type RoomService struct {
	mu      sync.RWMutex
	catalog models.RoomCatalog
}

// NewRoomService wraps a loaded room catalog.
func NewRoomService(catalog models.RoomCatalog) *RoomService {
	s := &RoomService{}
	s.Replace(catalog)
	return s
}

// Replace swaps in a freshly loaded catalog.
func (s *RoomService) Replace(catalog models.RoomCatalog) {
	if catalog.Rooms == nil {
		catalog.Rooms = make(map[string]models.Room)
	}
	s.mu.Lock()
	s.catalog = catalog
	s.mu.Unlock()
}

func (s *RoomService) current() models.RoomCatalog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.catalog
}

// All returns every room with a loaded layout, keyed by name. The map must
// not be modified.
func (s *RoomService) All() map[string]models.Room {
	return s.current().Rooms
}

// Find returns the rooms among names that exist, keyed by name.
func (s *RoomService) Find(names []string) map[string]models.Room {
	rooms := s.current().Rooms
	found := make(map[string]models.Room)
	for _, name := range names {
		if room, ok := rooms[name]; ok {
			found[name] = room
		}
	}
	return found
}

// Has reports whether a room has a loaded layout.
func (s *RoomService) Has(name string) bool {
	_, ok := s.current().Rooms[name]
	return ok
}

// Count is the number of rooms with a loaded layout.
func (s *RoomService) Count() int {
	return len(s.current().Rooms)
}

// ConfiguredCount is the number of rooms listed in metadata.json.
func (s *RoomService) ConfiguredCount() int {
	return s.current().Configured
}

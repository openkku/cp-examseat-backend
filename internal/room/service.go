package room

import "sync"

// Service serves the room catalog (layouts, images, map links). The catalog
// is reloaded at runtime when room files change.
type Service struct {
	loader  CatalogLoader
	mu      sync.RWMutex
	catalog Catalog
}

// NewService loads the catalog from loader.
func NewService(loader CatalogLoader) *Service {
	s := &Service{loader: loader}
	s.Reload()
	return s
}

// Reload re-reads the catalog from the loader.
func (s *Service) Reload() {
	s.Replace(s.loader.Load())
}

// Replace swaps in a freshly loaded catalog.
func (s *Service) Replace(catalog Catalog) {
	if catalog.Rooms == nil {
		catalog.Rooms = make(map[string]Room)
	}
	s.mu.Lock()
	s.catalog = catalog
	s.mu.Unlock()
}
func (s *Service) current() Catalog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.catalog
}

// All returns every room with a loaded layout, keyed by name. The map must
// not be modified.
func (s *Service) All() map[string]Room {
	return s.current().Rooms
}

// Find returns the rooms among names that exist, keyed by name.
func (s *Service) Find(names []string) map[string]Room {
	rooms := s.current().Rooms
	found := make(map[string]Room)
	for _, name := range names {
		if room, ok := rooms[name]; ok {
			found[name] = room
		}
	}
	return found
}

// Has reports whether a room has a loaded layout.
func (s *Service) Has(name string) bool {
	_, ok := s.current().Rooms[name]
	return ok
}

// Count is the number of rooms with a loaded layout.
func (s *Service) Count() int {
	return len(s.current().Rooms)
}

// ConfiguredCount is the number of rooms listed in metadata.json.
func (s *Service) ConfiguredCount() int {
	return s.current().Configured
}

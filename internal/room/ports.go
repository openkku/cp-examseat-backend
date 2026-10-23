package room

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

// CatalogLoader reads the room catalog from where it is stored.
//
// Implemented by room/filesystem.Loader.
type CatalogLoader interface {
	Load() Catalog
}

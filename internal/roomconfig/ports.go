package roomconfig

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

import "github.com/openkku/cp-examseat-backend/internal/room"

// Store reads and writes the stored room configuration. Layout file names may
// contain directory components; the store confines them to its layout folder.
//
// Implemented by roomconfig/filesystem.Store.
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

package datacheck

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/room"
)

// RoomSeats holds the distinct seat identifiers used in one exam round and room.
type RoomSeats struct {
	Round string
	Room  string
	Seats []string
}

// Usage queries actual seat assignments in the database.
//
// Implemented by database/sqlite.
type Usage interface {
	RoomSeats(ctx context.Context) ([]RoomSeats, error)
}

// Rooms accesses configured room metadata and seating layouts.
//
// Implemented by roomconfig.Service.
type Rooms interface {
	Rooms() (map[string]room.Meta, error)
	// RawLayout returns a layout file; exists is false when there is none.
	RawLayout(file string) (data []byte, exists bool, err error)
}

// Images checks for image file existence on disk.
type Images interface {
	Exists(webPath string) bool
}

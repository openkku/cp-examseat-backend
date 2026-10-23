package tui

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/room"
)

// Rounds is the round management the TUI drives.
//
// Implemented by admin.Service.
type Rounds interface {
	Rounds(ctx context.Context) ([]admin.RoundSummary, error)
	RenameRound(ctx context.Context, round, label string) (string, error)
	DeleteRound(ctx context.Context, round, customID string) error
	ValidateFileType(ext string) error
	Import(ctx context.Context, req admin.ImportRequest) (int, error)
	Reload(ctx context.Context) error
}

// Rooms is the room editing the TUI drives.
//
// Implemented by roomconfig.Service.
type Rooms interface {
	Rooms() (map[string]room.Meta, error)
	SaveRoom(roomID string, meta room.Meta) error
	DeleteRoom(roomID string) error
	Layout(filename string) ([]byte, error)
	SaveLayout(filename string, body []byte) (int, error)
}

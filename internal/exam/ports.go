package exam

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Repository reads stored seats.
//
// Implemented by database/sqlite.
type Repository interface {
	GetSeatsByID(ctx context.Context, q domain.SeatQuery) ([]domain.Seat, error)
	GetSeats(ctx context.Context, q ExploreQuery) ([]domain.Seat, error)
	GetOptions(ctx context.Context, q OptionsQuery) ([]string, error)
}

// Upstream makes sure a student's seats are stored before they are read, for
// rounds whose seats are fetched on demand (checkex). client identifies the
// visitor for per-client limits.
//
// Implemented by checkex.Service, or NoUpstream when checkex is off.
type Upstream interface {
	Ensure(ctx context.Context, round, studentID, client string) error
	EnsureAll(ctx context.Context, studentID, client string) error
}

// NoUpstream is the Upstream used when no round is fetched on demand.
type NoUpstream struct{}

// Ensure does nothing.
func (NoUpstream) Ensure(context.Context, string, string, string) error { return nil }

// EnsureAll does nothing.
func (NoUpstream) EnsureAll(context.Context, string, string) error { return nil }

// RoomDirectory tells which rooms have a seating layout.
//
// Implemented by room.Service.
type RoomDirectory interface {
	Has(name string) bool
	Count() int
}

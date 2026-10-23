// Package exam answers seat lookups: a student's schedule (also as a calendar
// feed), a room's roster and the explorer's cascading filter options.
package exam

import (
	"context"
	"errors"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Option modes accepted by OptionsQuery.Mode.
const (
	OptionDates = "dates"
	OptionTimes = "times"
	OptionRooms = "rooms"
)

// ErrInvalidOptionMode is returned for an OptionsQuery with an unknown Mode.
var ErrInvalidOptionMode = errors.New("invalid mode")

// ExploreQuery selects the roster of one room for one exam slot.
type ExploreQuery struct {
	Round string
	Room  string
	Date  string
	Time  string
	Seat  *string
}

// OptionsQuery lists the distinct dates, times or rooms of a round for the
// cascading explorer filters.
type OptionsQuery struct {
	Mode  string
	Round string
	Date  string
	Time  string
	// RoomFilter, when set, keeps only rows whose room it accepts
	// (used to hide rooms that have no seating layout).
	RoomFilter func(room string) bool
}

// AcceptsRoom reports whether a room passes the optional RoomFilter.
func (q OptionsQuery) AcceptsRoom(room string) bool {
	return q.RoomFilter == nil || q.RoomFilter(room)
}

// Repository reads stored seats.
type Repository interface {
	GetSeatsByID(ctx context.Context, q domain.SeatQuery) ([]domain.Seat, error)
	GetSeats(ctx context.Context, q ExploreQuery) ([]domain.Seat, error)
	GetOptions(ctx context.Context, q OptionsQuery) ([]string, error)
}

// Upstream makes sure a student's seats are stored before they are read, for
// rounds whose seats are fetched on demand (checkex). client identifies the
// visitor for per-client limits.
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
type RoomDirectory interface {
	Has(name string) bool
	Count() int
}

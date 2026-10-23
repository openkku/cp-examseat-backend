package models

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Upstream lookup outcomes, recorded per (round, student) so a student is
// fetched from checkex at most once.
const (
	LookupFound    = "found"
	LookupNotFound = "not_found"
)

// Lookup is the recorded outcome of one upstream (checkex) lookup.
type Lookup struct {
	Status    string
	FetchedAt time.Time
}

// LookupRepository persists seats fetched one student at a time from checkex
// and remembers which lookups were already made.
type LookupRepository interface {
	// GetSeatsByID is used to skip students whose seats were imported another way.
	GetSeatsByID(ctx context.Context, q SeatQuery) ([]Seat, error)
	// EnsureRound creates the round if it does not exist, keeping an existing label.
	EnsureRound(ctx context.Context, roundID string, label string) error
	// AddStudentSeats stores one student's seats and records the lookup as found.
	AddStudentSeats(ctx context.Context, roundID string, studentID string, seats []Seat) error
	// GetLookup returns the recorded lookup, or nil when the student was never looked up.
	GetLookup(ctx context.Context, roundID string, studentID string) (*Lookup, error)
	// MarkNotFound records that checkex has no seats for the student.
	MarkNotFound(ctx context.Context, roundID string, studentID string) error
}

// ErrLookupBusy means the scrapler queue is full; the visitor should retry later.
var ErrLookupBusy = errors.New("checkex: lookup queue is full")

// ErrLookupLimited means this visitor has caused too many not-found lookups, or
// has too many lookups waiting in the scrapler queue. Nothing was sent upstream.
var ErrLookupLimited = errors.New("checkex: too many lookups from this client")

// LookupQueuedError means scrapler didn't finish the lookup in time. It keeps
// running there and its answer is kept for a while, so retrying later is served
// without a new upstream call.
type LookupQueuedError struct {
	// Position is how many lookups are ahead of this one; 0 means it is running now.
	Position int
}

func (e *LookupQueuedError) Error() string {
	return fmt.Sprintf("checkex: lookup queued (%d ahead)", e.Position)
}

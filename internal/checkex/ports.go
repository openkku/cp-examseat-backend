package checkex

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Repository persists seats fetched one student at a time from checkex
// and remembers which lookups were already made.
//
// Implemented by database/sqlite.
type Repository interface {
	// HasRegularSeats reports whether the student already has seats in the
	// round outside custom datasets, i.e. seats imported another way.
	HasRegularSeats(ctx context.Context, roundID string, studentID string) (bool, error)
	// EnsureRound creates the round if it does not exist, keeping an existing label.
	EnsureRound(ctx context.Context, roundID string, label string) error
	// AddStudentSeats stores one student's seats and records the lookup as found.
	AddStudentSeats(ctx context.Context, roundID string, studentID string, seats []domain.Seat) error
	// GetLookup returns the recorded lookup, or nil when the student was never looked up.
	GetLookup(ctx context.Context, roundID string, studentID string) (*Lookup, error)
	// MarkNotFound records that checkex has no seats for the student.
	MarkNotFound(ctx context.Context, roundID string, studentID string) error
}

// Source fetches one student's seats from upstream (the scrapler client).
// found is false when checkex has no seats for the ID.
//
// Implemented by checkex/scrapler.Client.
type Source interface {
	Lookup(ctx context.Context, roundID, studentID, client string) (seats []domain.Seat, found bool, err error)
}

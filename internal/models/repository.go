package models

import (
	"context"
	"errors"
)

// ErrInvalidOptionMode is returned for an OptionsQuery with an unknown Mode.
var ErrInvalidOptionMode = errors.New("invalid mode")

// ExamRepository persists exam rounds and seats.
type ExamRepository interface {
	GetSeatsByID(ctx context.Context, q SeatQuery) ([]Seat, error)
	GetSeats(ctx context.Context, q ExploreQuery) ([]Seat, error)
	GetOptions(ctx context.Context, q OptionsQuery) ([]string, error)

	AddRound(ctx context.Context, roundID string, displayName string, seats []Seat) error
	PurgeRound(ctx context.Context, roundID string) error
	PurgeCustomDataset(ctx context.Context, roundID string, customID string) error
	GetRounds(ctx context.Context) ([]Round, error)
	GetAllSeats(ctx context.Context) ([]Seat, error)
}

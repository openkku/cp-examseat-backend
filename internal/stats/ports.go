package stats

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Repository reads every stored seat.
//
// Implemented by database/sqlite.
type Repository interface {
	GetAllSeats(ctx context.Context) ([]domain.Seat, error)
}

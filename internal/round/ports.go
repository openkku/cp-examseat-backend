package round

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Repository reads the stored rounds.
//
// Implemented by database/sqlite.
type Repository interface {
	GetRounds(ctx context.Context) ([]domain.Round, error)
}

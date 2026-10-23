package sqlite

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// GetRounds lists every round, newest first.
func (d *DB) GetRounds(ctx context.Context) ([]domain.Round, error) {
	return d.rounds(ctx)
}

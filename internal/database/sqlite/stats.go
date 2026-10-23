package sqlite

import (
	"context"
	"fmt"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// GetAllSeats returns every seat of every round (used to build analytics).
func (d *DB) GetAllSeats(ctx context.Context) ([]domain.Seat, error) {
	rows, err := d.db.QueryContext(ctx, seatSelect)
	if err != nil {
		return nil, fmt.Errorf("failed to query all seats: %w", err)
	}
	defer rows.Close()

	return scanSeats(rows)
}

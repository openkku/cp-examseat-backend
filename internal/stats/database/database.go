// Package database reads the seats analysed by the stats slice from the SQL database.
package database

import (
	"context"
	"fmt"

	platformdb "github.com/openkku/cp-examseat-backend/internal/platform/database"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
	"github.com/openkku/cp-examseat-backend/internal/stats"
)

// Repo implements stats.SeatSource.
type Repo struct {
	db *platformdb.DB
}

var _ stats.SeatSource = (*Repo)(nil)

// New creates a Repo on db.
func New(db *platformdb.DB) *Repo {
	return &Repo{db: db}
}

// GetAllSeats returns every seat of every round (used to build analytics).
func (r *Repo) GetAllSeats(ctx context.Context) ([]domain.Seat, error) {
	rows, err := r.db.SQL().QueryContext(ctx, platformdb.SeatSelect)
	if err != nil {
		return nil, fmt.Errorf("failed to query all seats: %w", err)
	}
	defer rows.Close()

	return platformdb.ScanSeats(rows)
}

// Package database stores rounds in the SQL database.
package database

import (
	"context"

	platformdb "github.com/openkku/cp-examseat-backend/internal/platform/database"
	"github.com/openkku/cp-examseat-backend/internal/round"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Repo implements round.Repository.
type Repo struct {
	db *platformdb.DB
}

var _ round.Repository = (*Repo)(nil)

// New creates a Repo on db.
func New(db *platformdb.DB) *Repo {
	return &Repo{db: db}
}

// GetRounds lists every round, newest first.
func (r *Repo) GetRounds(ctx context.Context) ([]domain.Round, error) {
	return platformdb.Rounds(ctx, r.db.SQL())
}

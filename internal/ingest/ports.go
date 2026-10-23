package ingest

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Extractor reads the seats of a seating file.
//
// Implemented by ingest/importer.Registry.
type Extractor interface {
	// Supports reports whether files with the (lower-case, dotted) extension can be read.
	Supports(ext string) bool
	Extract(filePath string, p Params) ([]domain.Seat, error)
}

// Repository replaces stored seats.
//
// Implemented by database/sqlite.
type Repository interface {
	PurgeRound(ctx context.Context, roundID string) error
	PurgeCustomDataset(ctx context.Context, roundID string, customID string) error
	AddRound(ctx context.Context, roundID string, displayName string, seats []domain.Seat) error
}

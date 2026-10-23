package admin

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Repository manages stored rounds. Operations on an unknown round return
// domain.ErrRoundNotFound.
//
// Implemented by database/sqlite.
type Repository interface {
	RoundSummaries(ctx context.Context) ([]RoundSummary, error)
	RoundExists(ctx context.Context, roundID string) (bool, error)
	SetRoundLabel(ctx context.Context, roundID string, label string) error
	DeleteRound(ctx context.Context, roundID string) error
}

// DatasetPurger removes one custom dataset of a round.
//
// Implemented by database/sqlite.
type DatasetPurger interface {
	PurgeCustomDataset(ctx context.Context, roundID string, customID string) error
}

// Importer imports seating files.
//
// Implemented by ingest.Service.
type Importer interface {
	SupportedExtension(ext string) bool
	Ingest(ctx context.Context, opts ingest.Options) (int, error)
	Preview(opts ingest.Options) ([]domain.Seat, error)
}

// Reloader asks the running server to refresh its in-memory state (rounds,
// rooms, statistics, caches) after data changes.
//
// Implemented by platform/reloadsignal.Notifier.
type Reloader interface {
	Reload(ctx context.Context) error
}

package admin

// Ports: what this slice needs from the outside. Each is implemented by an
// adapter (or another slice's core) and wired in internal/app.

import (
	"context"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/ingest"
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
}

// Rooms counts the rooms of the catalog.
//
// Implemented by room.Service.
type Rooms interface {
	Count() int
	ConfiguredCount() int
}

// Reloader refreshes the in-memory state (rounds, rooms, statistics, caches)
// after data changes.
//
// Implemented by app.App.
type Reloader interface {
	Reload(ctx context.Context) error
	LastReload() time.Time
}

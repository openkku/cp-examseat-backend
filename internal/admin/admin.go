// Package admin implements the token-protected administration used by the
// frontend's admin page: importing seating files, renaming and deleting
// rounds and custom datasets, and reloading the in-memory state.
package admin

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// CustomDataset is an out-of-schedule dataset imported under a custom ID.
type CustomDataset struct {
	ID    string `json:"id"`
	Seats int    `json:"seats"`
}

// RoundSummary describes the data stored for one round.
type RoundSummary struct {
	ID              string          `json:"id"`
	Label           string          `json:"label"`
	Seats           int             `json:"seats"`
	Students        int             `json:"students"`
	InScheduleSeats int             `json:"in_schedule_seats"`
	CustomDatasets  []CustomDataset `json:"custom_datasets"`
}

// Status is the state shown on the admin page.
type Status struct {
	Rooms           int
	RoomsConfigured int
	LastReload      time.Time
}

// Repository manages stored rounds. Operations on an unknown round return
// domain.ErrRoundNotFound.
type Repository interface {
	RoundSummaries(ctx context.Context) ([]RoundSummary, error)
	RoundExists(ctx context.Context, roundID string) (bool, error)
	SetRoundLabel(ctx context.Context, roundID string, label string) error
	DeleteRound(ctx context.Context, roundID string) error
}

// DatasetPurger removes one custom dataset of a round.
type DatasetPurger interface {
	PurgeCustomDataset(ctx context.Context, roundID string, customID string) error
}

// Importer imports seating files.
type Importer interface {
	SupportedExtension(ext string) bool
	Ingest(ctx context.Context, opts ingest.Options) (int, error)
}

// Rooms counts the rooms of the catalog.
type Rooms interface {
	Count() int
	ConfiguredCount() int
}

// Reloader refreshes the in-memory state (rounds, rooms, statistics, caches)
// after data changes.
type Reloader interface {
	Reload(ctx context.Context) error
	LastReload() time.Time
}

// InvalidInputError is a request the admin must correct; Message says how.
type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string { return e.Message }

func invalid(message string) error { return &InvalidInputError{Message: message} }

// ReloadError means a change was saved but reloading the in-memory state failed.
type ReloadError struct {
	Err error
}

func (e *ReloadError) Error() string { return e.Err.Error() }

func (e *ReloadError) Unwrap() error { return e.Err }

// idPattern restricts round and dataset IDs to safe identifiers.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_\-]{1,64}$`)

// maxLabelLength bounds a round label.
const maxLabelLength = 200

// ImportRequest is one seating file to import.
type ImportRequest struct {
	FilePath   string
	Round      string
	Display    string
	Labels     []string
	RoomLayout string
	CustomID   string
}

// Service implements the admin use cases.
type Service struct {
	repo     Repository
	datasets DatasetPurger
	importer Importer
	rooms    Rooms
	reloader Reloader
}

// NewService creates a Service.
func NewService(repo Repository, datasets DatasetPurger, importer Importer, rooms Rooms, reloader Reloader) *Service {
	return &Service{repo: repo, datasets: datasets, importer: importer, rooms: rooms, reloader: reloader}
}

// Status reports the loaded rooms and when data was last reloaded.
func (s *Service) Status() Status {
	return Status{
		Rooms:           s.rooms.Count(),
		RoomsConfigured: s.rooms.ConfiguredCount(),
		LastReload:      s.reloader.LastReload(),
	}
}

// LastReload is when data was last (re)loaded.
func (s *Service) LastReload() time.Time {
	return s.reloader.LastReload()
}

// Rounds lists every round with its seat and student counts, newest first.
func (s *Service) Rounds(ctx context.Context) ([]RoundSummary, error) {
	return s.repo.RoundSummaries(ctx)
}

// RenameRound sets the label of a round and returns the stored label.
func (s *Service) RenameRound(ctx context.Context, round, label string) (string, error) {
	label = strings.TrimSpace(label)
	if !idPattern.MatchString(round) || label == "" || len(label) > maxLabelLength {
		return "", invalid("A round ID and a label of at most 200 characters are required")
	}
	if err := s.repo.SetRoundLabel(ctx, round, label); err != nil {
		return "", err
	}
	return label, s.Reload(ctx)
}

// DeleteRound removes a whole round, or only its custom dataset customID when set.
func (s *Service) DeleteRound(ctx context.Context, round, customID string) error {
	if !idPattern.MatchString(round) || (customID != "" && !idPattern.MatchString(customID)) {
		return invalid("Invalid round or dataset ID")
	}
	// A failed check lets the delete itself report the database error.
	if exists, err := s.repo.RoundExists(ctx, round); err == nil && !exists {
		return domain.ErrRoundNotFound
	}

	var err error
	if customID != "" {
		err = s.datasets.PurgeCustomDataset(ctx, round, customID)
	} else {
		err = s.repo.DeleteRound(ctx, round)
	}
	if err != nil {
		return err
	}
	return s.Reload(ctx)
}

// ValidateImport checks the identifiers of an import before its file is read.
func (s *Service) ValidateImport(round, customID, roomLayout string) error {
	if !idPattern.MatchString(round) {
		return invalid("Round ID is required (letters, digits, _ and - only)")
	}
	if (customID != "" && !idPattern.MatchString(customID)) || (roomLayout != "" && !idPattern.MatchString(roomLayout)) {
		return invalid("Invalid custom ID or room layout")
	}
	return nil
}

// ValidateFileType checks that a file with extension ext (lower-case, dotted) can be imported.
func (s *Service) ValidateFileType(ext string) error {
	if !s.importer.SupportedExtension(ext) {
		return invalid("Unsupported file type (use .xlsx, .xls, .pdf or .json)")
	}
	return nil
}

// Import imports a seating file and returns the number of seats saved. An
// empty Display defaults to the round's derived label.
func (s *Service) Import(ctx context.Context, req ImportRequest) (int, error) {
	if err := s.ValidateImport(req.Round, req.CustomID, req.RoomLayout); err != nil {
		return 0, err
	}
	display := req.Display
	if display == "" {
		display = domain.DefaultRoundLabel(req.Round)
	}

	saved, err := s.importer.Ingest(ctx, ingest.Options{
		FilePath:    req.FilePath,
		RoundID:     req.Round,
		DisplayName: display,
		Labels:      req.Labels,
		RoomLayout:  req.RoomLayout,
		CustomID:    req.CustomID,
	})
	if err != nil {
		return 0, err
	}
	return saved, s.Reload(ctx)
}

// Reload re-reads the in-memory state; a failure is returned as *ReloadError.
func (s *Service) Reload(ctx context.Context) error {
	if err := s.reloader.Reload(ctx); err != nil {
		return &ReloadError{Err: err}
	}
	return nil
}

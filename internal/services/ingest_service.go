package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/importers/jsonfile"
	"github.com/openkku/cp-examseat-backend/internal/importers/pdf"
	"github.com/openkku/cp-examseat-backend/internal/importers/xlsx"
	"github.com/openkku/cp-examseat-backend/internal/models"
)

// IngestOptions describes one seating file to import.
type IngestOptions struct {
	FilePath    string
	RoundID     string
	DisplayName string
	Labels      []string
	RoomLayout  string
	CustomID    string
}

// IngestService imports seating files (.pdf, .xlsx/.xls, .json) into the repository.
type IngestService struct {
	repo models.ExamRepository
}

// NewIngestService creates an IngestService writing to repo.
func NewIngestService(repo models.ExamRepository) *IngestService {
	return &IngestService{repo: repo}
}

// Ingest extracts the seats of opts.FilePath and replaces the matching data:
// the custom datasets it contains when any seat carries a custom ID, otherwise
// the round's in-schedule sessions. It returns the number of seats saved.
func (s *IngestService) Ingest(ctx context.Context, opts IngestOptions) (int, error) {
	fmt.Printf("📦 Ingesting file [%s] for round '%s' (Custom ID: '%s', Labels: %v)\n", opts.FilePath, opts.RoundID, opts.CustomID, opts.Labels)

	seats, err := extractSeats(opts)
	if err != nil {
		return 0, err
	}

	// Collect unique custom_ids from options and individual seats (if ingesting JSON)
	customIDsToPurge := make(map[string]bool)
	if opts.CustomID != "" {
		customIDsToPurge[opts.CustomID] = true
	}
	for _, seat := range seats {
		if seat.CustomID != "" {
			customIDsToPurge[seat.CustomID] = true
		}
	}

	if len(customIDsToPurge) > 0 {
		for cid := range customIDsToPurge {
			fmt.Printf("   > Purging old entries for round '%s' and custom_id '%s'...\n", opts.RoundID, cid)
			if err := s.repo.PurgeCustomDataset(ctx, opts.RoundID, cid); err != nil {
				return 0, fmt.Errorf("failed to purge custom dataset '%s': %w", cid, err)
			}
		}
	} else {
		fmt.Printf("   > Purging old entries for round '%s'...\n", opts.RoundID)
		if err := s.repo.PurgeRound(ctx, opts.RoundID); err != nil {
			return 0, fmt.Errorf("failed to purge round: %w", err)
		}
	}

	fmt.Printf("   > Saving %d seats into database...\n", len(seats))
	if err := s.repo.AddRound(ctx, opts.RoundID, opts.DisplayName, seats); err != nil {
		return 0, fmt.Errorf("failed to save round seats: %w", err)
	}

	fmt.Printf("✅ Ingestion complete! Successfully saved %d seats for round '%s'.\n", len(seats), opts.RoundID)
	return len(seats), nil
}

func extractSeats(opts IngestOptions) ([]models.Seat, error) {
	var seats []models.Seat
	var err error

	switch ext := strings.ToLower(filepath.Ext(opts.FilePath)); ext {
	case ".pdf":
		seats, err = pdf.ExtractSeats(opts.FilePath, opts.RoundID, opts.Labels, opts.RoomLayout, opts.CustomID)
	case ".xlsx", ".xls":
		seats, err = xlsx.ExtractSeats(opts.FilePath, opts.RoundID, opts.Labels, opts.RoomLayout, opts.CustomID)
	case ".json":
		seats, err = jsonfile.ExtractSeats(opts.FilePath, opts.RoundID, opts.Labels, opts.RoomLayout, opts.CustomID)
	default:
		return nil, fmt.Errorf("unsupported file extension %q (supported: .pdf, .xlsx, .json)", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to extract seats from %s: %w", opts.FilePath, err)
	}
	return seats, nil
}

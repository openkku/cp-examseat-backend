package ingest

import (
	"context"
	"fmt"
	"strings"
)

// Service imports seating files into a Repository.
type Service struct {
	repo      Repository
	extractor Extractor
}

// NewService creates a Service writing to repo.
func NewService(repo Repository, extractor Extractor) *Service {
	return &Service{repo: repo, extractor: extractor}
}

// SupportedExtension reports whether files with the (lower-case, dotted) extension can be imported.
func (s *Service) SupportedExtension(ext string) bool {
	return s.extractor.Supports(ext)
}

// Ingest extracts the seats of opts.FilePath and replaces the matching data:
// the custom datasets it contains when any seat carries a custom ID, otherwise
// the round's in-schedule sessions. It returns the number of seats saved.
func (s *Service) Ingest(ctx context.Context, opts Options) (int, error) {
	fmt.Printf("📦 Ingesting file [%s] for round '%s' (Custom ID: '%s', Labels: %v)\n", opts.FilePath, opts.RoundID, opts.CustomID, opts.Labels)

	seats, err := s.extractor.Extract(opts.FilePath, Params{
		RoundID:    opts.RoundID,
		Labels:     opts.Labels,
		RoomLayout: opts.RoomLayout,
		CustomID:   opts.CustomID,
	})
	if err != nil {
		return 0, err
	}
	// Labels are read back comma-joined, so a comma would split one label in two.
	// Checked before purging so a rejected file leaves the stored data intact.
	for _, seat := range seats {
		for _, label := range seat.Labels {
			if strings.Contains(label, ",") {
				return 0, fmt.Errorf("label %q contains a comma", label)
			}
		}
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

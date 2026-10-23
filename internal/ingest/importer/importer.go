// Package importer picks the seat extractor (pdf, xlsx, jsonfile) for a
// seating file by its extension.
package importer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/ingest/importer/jsonfile"
	"github.com/openkku/cp-examseat-backend/internal/ingest/importer/pdf"
	"github.com/openkku/cp-examseat-backend/internal/ingest/importer/xlsx"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

type extractFunc func(filePath string, roundID string, labels []string, roomLayout string, customID string) ([]domain.Seat, error)

var byExtension = map[string]extractFunc{
	".pdf":  pdf.ExtractSeats,
	".xlsx": xlsx.ExtractSeats,
	".xls":  xlsx.ExtractSeats,
	".json": jsonfile.ExtractSeats,
}

// Registry implements ingest.Extractor with the built-in file formats.
type Registry struct{}

var _ ingest.Extractor = Registry{}

// Supports reports whether ext (lower-case, with the dot) is a known format.
func (Registry) Supports(ext string) bool {
	_, ok := byExtension[ext]
	return ok
}

// Extract reads the seats of filePath with the extractor of its extension.
func (Registry) Extract(filePath string, p ingest.Params) ([]domain.Seat, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	extract, ok := byExtension[ext]
	if !ok {
		return nil, fmt.Errorf("unsupported file extension %q (supported: .pdf, .xlsx, .json)", ext)
	}
	seats, err := extract(filePath, p.RoundID, p.Labels, p.RoomLayout, p.CustomID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract seats from %s: %w", filePath, err)
	}
	return seats, nil
}

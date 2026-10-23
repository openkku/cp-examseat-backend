// Package jsonfile imports exam seats from JSON files such as the output of
// script/parse_enrollment.py (an array of objects keyed by domain.Seat field names).
package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// ExtractSeats reads seats from a JSON file, filling blanks with the given
// defaults and deriving TimeStart/TimeEnd and Category.
func ExtractSeats(filePath string, roundID string, defaultLabels []string, roomLayout string, customID string) ([]domain.Seat, error) {
	data, err := os.ReadFile(filePath) // #nosec G304 -- operator-supplied import file
	if err != nil {
		return nil, fmt.Errorf("failed to read json file: %w", err)
	}

	var seats []domain.Seat
	if err := json.Unmarshal(data, &seats); err != nil {
		return nil, fmt.Errorf("failed to parse json seats: %w", err)
	}

	for i := range seats {
		s := &seats[i]
		if s.ExamRound == "" {
			s.ExamRound = roundID
		}
		if len(s.Labels) == 0 && len(defaultLabels) > 0 {
			s.Labels = defaultLabels
		}
		if s.RoomLayout == "" && roomLayout != "" {
			s.RoomLayout = roomLayout
		}
		if s.CustomID == "" && customID != "" {
			s.CustomID = customID
		}
		if s.TimeStart == "" && s.Time != "" {
			s.TimeStart, s.TimeEnd = domain.SplitTime(s.Time)
		}
		if s.Category == "" {
			if s.CustomID != "" {
				s.Category = domain.CategoryOutOfSchedule
			} else {
				s.Category = domain.CategoryInSchedule
			}
		}
	}

	return seats, nil
}

package jsonfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSeats(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "mock_schedule.json")

	jsonContent := `[
		{
			"sheet": "Sheet1",
			"date": "2026-09-15",
			"time": "09.00 - 12.00",
			"room": "CP9421",
			"subject": "CP422021",
			"subject_name": "Web & Mobile Architecture",
			"section": "1",
			"student_id": "683380001-1",
			"seat": "A01"
		}
	]`

	if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write mock json: %v", err)
	}

	seats, err := ExtractSeats(jsonFile, "mid_1_2569", []string{"Lecture", "นัดสอบนอกตาราง"}, "CP9421_LAYOUT", "MID_1_2569_CP422021_LEC")
	if err != nil {
		t.Fatalf("ExtractSeats failed: %v", err)
	}

	if len(seats) != 1 {
		t.Fatalf("Expected 1 seat parsed, got %d", len(seats))
	}

	s := seats[0]
	if s.ExamRound != "mid_1_2569" {
		t.Errorf("Expected round 'mid_1_2569', got %s", s.ExamRound)
	}
	if s.CustomID != "MID_1_2569_CP422021_LEC" {
		t.Errorf("Expected custom_id 'MID_1_2569_CP422021_LEC', got %s", s.CustomID)
	}
	if len(s.Labels) != 2 || s.Labels[0] != "Lecture" || s.Labels[1] != "นัดสอบนอกตาราง" {
		t.Errorf("Expected labels ['Lecture', 'นัดสอบนอกตาราง'], got %v", s.Labels)
	}
	if s.RoomLayout != "CP9421_LAYOUT" {
		t.Errorf("Expected room_layout 'CP9421_LAYOUT', got %s", s.RoomLayout)
	}
	if s.TimeStart != "09.00" || s.TimeEnd != "12.00" {
		t.Errorf("Expected time_start 09.00 and time_end 12.00, got %s / %s", s.TimeStart, s.TimeEnd)
	}
	if s.Category != "OUT_OF_SCHEDULE" {
		t.Errorf("Expected category OUT_OF_SCHEDULE, got %s", s.Category)
	}
}

func TestExtractSeatsGoFieldNames(t *testing.T) {
	jsonFile := filepath.Join(t.TempDir(), "enrollment.json")
	content := `[{"Sheet": "S", "Date": "2026-09-01", "Time": "13.00", "Room": "CP9127", "Subject": "CP1", "StudentID": "6633801326", "Seat": "A1", "Labels": ["Lab"]}]`
	if err := os.WriteFile(jsonFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write json: %v", err)
	}

	seats, err := ExtractSeats(jsonFile, "mid_1_2569", []string{"Default"}, "", "")
	if err != nil {
		t.Fatalf("ExtractSeats failed: %v", err)
	}
	s := seats[0]
	if s.StudentID != "6633801326" || s.Room != "CP9127" {
		t.Errorf("Go field names were not mapped: %+v", s)
	}
	if len(s.Labels) != 1 || s.Labels[0] != "Lab" {
		t.Errorf("Expected explicit labels to win over defaults, got %v", s.Labels)
	}
	if s.TimeStart != "13.00" || s.TimeEnd != "" {
		t.Errorf("Expected open-ended time 13.00, got %q-%q", s.TimeStart, s.TimeEnd)
	}
	if s.Category != "IN_SCHEDULE" {
		t.Errorf("Expected IN_SCHEDULE without custom id, got %s", s.Category)
	}
}

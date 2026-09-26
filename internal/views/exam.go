package views

import "github.com/openkku/cp-examseat-backend/internal/models"

// ExamSchedule is the public JSON representation of a seat.
type ExamSchedule struct {
	Sheet       string   `json:"sheet"`
	ExamRound   string   `json:"exam_round,omitempty"`
	Category    string   `json:"category,omitempty"`
	Date        string   `json:"date"`
	Time        string   `json:"time"`
	TimeStart   string   `json:"time_start,omitempty"`
	TimeEnd     string   `json:"time_end,omitempty"`
	Room        string   `json:"room"`
	Subject     string   `json:"subject"`
	SubjectName string   `json:"subject_name"`
	Section     string   `json:"section"`
	StudentID   string   `json:"student_id"`
	Seat        string   `json:"seat"`
	Note        string   `json:"note"`
	Branch      string   `json:"branch"`
	Labels      []string `json:"labels,omitempty"`
	RoomLayout  string   `json:"room_layout,omitempty"`
	CustomID    string   `json:"custom_id,omitempty"`
}

// NewExamSchedule maps a seat to its public representation. Internal fields
// (round, category, split start/end time) are intentionally left out.
func NewExamSchedule(s models.Seat) ExamSchedule {
	return ExamSchedule{
		Sheet:       s.Sheet,
		Date:        s.Date,
		Time:        s.Time,
		Room:        s.Room,
		Subject:     s.Subject,
		SubjectName: s.SubjectName,
		Section:     s.Section,
		StudentID:   s.StudentID,
		Seat:        s.Seat,
		Note:        s.Note,
		Branch:      s.Branch,
		Labels:      s.Labels,
		RoomLayout:  s.RoomLayout,
		CustomID:    s.CustomID,
	}
}

// ExamSchedules maps seats to their public representation (never nil).
func ExamSchedules(seats []models.Seat) []ExamSchedule {
	out := make([]ExamSchedule, 0, len(seats))
	for _, s := range seats {
		out = append(out, NewExamSchedule(s))
	}
	return out
}

// Package domain is the shared kernel: the entities and rules every feature
// slice uses (seats, rounds, labels, student IDs, Thai dates). It depends on
// nothing but the standard library and knows nothing of HTTP or storage.
package domain

import "strings"

// Exam session categories stored with every seat.
const (
	CategoryInSchedule    = "IN_SCHEDULE"
	CategoryOutOfSchedule = "OUT_OF_SCHEDULE"
)

// LabelOutOfSchedule marks an exam arranged outside the official timetable.
const LabelOutOfSchedule = "นัดสอบนอกตาราง"

// Seat is one student's seat in one exam session.
//
// Field names intentionally carry no JSON tags: JSON import files (see
// script/parse_enrollment.py) use these Go field names as keys.
type Seat struct {
	Sheet       string
	Date        string
	Time        string
	TimeStart   string
	TimeEnd     string
	Category    string
	Room        string
	Subject     string
	SubjectName string
	Section     string
	StudentID   string
	Seat        string
	Note        string
	ExamRound   string
	Branch      string
	Labels      []string
	RoomLayout  string
	CustomID    string
}

// GetDate returns the exam date (YYYY-MM-DD).
func (s *Seat) GetDate() string {
	return s.Date
}

// GetTime returns the display time range, preferring TimeStart/TimeEnd over Time.
func (s *Seat) GetTime() string {
	if s.TimeStart != "" {
		if s.TimeEnd != "" {
			return s.TimeStart + "-" + s.TimeEnd
		}
		return s.TimeStart
	}
	return s.Time
}

// SplitTime splits a "HH.MM-HH.MM" range into its trimmed start and end parts.
// A value without a dash is returned as the start with an empty end.
func SplitTime(raw string) (start, end string) {
	parts := strings.Split(raw, "-")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(raw), ""
}

// ResolveCategory returns the seat category, inferring it when unset: seats
// from a custom dataset or labelled out-of-schedule are OUT_OF_SCHEDULE.
func (s *Seat) ResolveCategory() string {
	if s.Category != "" {
		return s.Category
	}
	if s.CustomID != "" || HasLabel(s.Labels, LabelOutOfSchedule) {
		return CategoryOutOfSchedule
	}
	return CategoryInSchedule
}

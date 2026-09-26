package views

import (
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

func TestRenderCalendar(t *testing.T) {
	seats := []models.Seat{
		{
			StudentID:   "653380123-4",
			Date:        "2026-06-10",
			Time:        "09.00 - 12.00",
			Subject:     "CP001001",
			SubjectName: "Introduction to Computer Science",
			Section:     "1",
			Room:        "Building 3 Room 301",
			Seat:        "A1",
			ExamRound:   "midterm",
		},
	}

	icsBytes, err := RenderCalendar(seats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	icsStr := string(icsBytes)
	expectedName := "NAME:CP KKU Exam Seats (653380123-4)"
	expectedCalName := "X-WR-CALNAME:CP KKU Exam Seats (653380123-4)"
	expectedRefresh := "REFRESH-INTERVAL:PT12H"

	if !strings.Contains(icsStr, expectedName) {
		t.Errorf("expected calendar to contain %q, but got:\n%s", expectedName, icsStr)
	}
	if !strings.Contains(icsStr, expectedCalName) {
		t.Errorf("expected calendar to contain %q, but got:\n%s", expectedCalName, icsStr)
	}
	if !strings.Contains(icsStr, expectedRefresh) {
		t.Errorf("expected calendar to contain %q, but got:\n%s", expectedRefresh, icsStr)
	}
}

func TestRenderCalendar_NoEndTime(t *testing.T) {
	seats := []models.Seat{
		{
			StudentID:   "663380132-6",
			Date:        "2026-08-30",
			Time:        "13.00",
			Subject:     "CP422022",
			SubjectName: "Database Architecture Analysis and Design",
			Section:     "1",
			Room:        "แจ้งก่อนวันสอบ",
			Seat:        "แจ้งก่อนวันสอบ",
			ExamRound:   "mid_1_2569",
		},
	}

	icsBytes, err := RenderCalendar(seats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	icsStr := string(icsBytes)
	if !strings.Contains(icsStr, "DTSTART") {
		t.Errorf("expected calendar to contain DTSTART, got:\n%s", icsStr)
	}
	if strings.Contains(icsStr, "DTEND") {
		t.Errorf("expected calendar NOT to contain DTEND for open-ended event, got:\n%s", icsStr)
	}
}

func TestRenderCalendar_Empty(t *testing.T) {
	icsBytes, err := RenderCalendar(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	icsStr := string(icsBytes)
	expectedName := "NAME:CP KKU Exam Seats"
	expectedCalName := "X-WR-CALNAME:CP KKU Exam Seats"
	expectedRefresh := "REFRESH-INTERVAL:PT12H"

	if !strings.Contains(icsStr, expectedName) {
		t.Errorf("expected calendar to contain %q, but got:\n%s", expectedName, icsStr)
	}
	if !strings.Contains(icsStr, expectedCalName) {
		t.Errorf("expected calendar to contain %q, but got:\n%s", expectedCalName, icsStr)
	}
	if !strings.Contains(icsStr, expectedRefresh) {
		t.Errorf("expected calendar to contain %q, but got:\n%s", expectedRefresh, icsStr)
	}
}

func TestRenderCalendar_TimeStartAndEnd(t *testing.T) {
	seats := []models.Seat{
		{
			StudentID:   "663380100-0",
			Date:        "2026-09-15",
			TimeStart:   "08.30",
			TimeEnd:     "11.30",
			Subject:     "CP422021",
			SubjectName: "Web Architectures",
			Section:     "2",
			Room:        "CP9421",
			Seat:        "B12",
			ExamRound:   "mid_1_2569",
			Labels:      []string{"นัดสอบนอกตาราง", "Lecture"},
			Note:        "นำเครื่องคิดเลขเข้าห้องสอบได้",
		},
	}

	icsBytes, err := RenderCalendar(seats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unfolded := strings.ReplaceAll(strings.ReplaceAll(string(icsBytes), "\r\n ", ""), "\n ", "")
	if !strings.Contains(unfolded, "DTSTART") || !strings.Contains(unfolded, "DTEND") {
		t.Errorf("expected calendar to contain both DTSTART and DTEND, got:\n%s", unfolded)
	}
	if !strings.Contains(unfolded, "นัดสอบนอกตาราง") {
		t.Errorf("expected calendar description to contain label 'นัดสอบนอกตาราง', got:\n%s", unfolded)
	}
	if !strings.Contains(unfolded, "นำเครื่องคิดเลขเข้าห้องสอบได้") {
		t.Errorf("expected calendar description to contain note, got:\n%s", unfolded)
	}
}

func TestRenderCalendar_OutOfScheduleEmptyTime(t *testing.T) {
	seats := []models.Seat{
		{
			StudentID:   "683380010-2",
			Date:        "2025-10-27",
			Time:        "",
			TimeStart:   "",
			TimeEnd:     "",
			Subject:     "CP421011",
			SubjectName: "Inspiration in Cybersecurity Career",
			Section:     "01",
			Room:        "จัดสอบนอกตาราง",
			Seat:        "จัดสอบนอกตาราง",
			ExamRound:   "final_1_2568",
			Labels:      []string{"นัดสอบนอกตาราง"},
		},
	}

	icsBytes, err := RenderCalendar(seats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	unfolded := strings.ReplaceAll(strings.ReplaceAll(string(icsBytes), "\r\n ", ""), "\n ", "")
	if strings.Contains(unfolded, "CP421011") {
		t.Errorf("expected event for CP421011 without exam time to be skipped from calendar feed, got:\n%s", unfolded)
	}
}

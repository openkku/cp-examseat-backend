// Package pdf imports exam seats from roster PDFs (requires poppler's pdftotext).
package pdf

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// ExtractSeats parses a PDF file using pdftotext and returns extracted seats.
func ExtractSeats(filePath string, defaultRound string, defaultLabels []string, roomLayout string, customID string) ([]models.Seat, error) {
	// An absolute path can never start with "-" and be read as an option.
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", filePath, err)
	}
	out, err := exec.Command("pdftotext", absPath, "-").Output() // #nosec G204 -- fixed binary, no shell; path is operator-supplied
	if err != nil {
		return nil, fmt.Errorf("failed to run pdftotext on %s: %w", filePath, err)
	}

	text := string(out)
	return ParsePDFText(text, defaultRound, defaultLabels, roomLayout, customID)
}

// ParsePDFText parses extracted text from a roster PDF into []models.Seat
func ParsePDFText(text string, defaultRound string, defaultLabels []string, roomLayout string, customID string) ([]models.Seat, error) {
	// Parse Subject Code & Subject Name
	// e.g. "รายวิชา CP421024 การเขียนโปรแกรมเชิงวัตถุ"
	var subject, subjectName, section, room, dateStr, timeStr string

	reSub := regexp.MustCompile(`รายวิชา\s+([A-Z0-9]+)\s+(.+)`)
	if match := reSub.FindStringSubmatch(text); len(match) > 2 {
		subject = strings.TrimSpace(match[1])
		subjectName = strings.TrimSpace(match[2])
		// Strip extra units info if present, e.g. "หน่วยกิต 3 (2-2-5)"
		if idx := strings.Index(subjectName, "หน่วยกิต"); idx != -1 {
			subjectName = strings.TrimSpace(subjectName[:idx])
		}
	}

	// Parse Section
	// e.g. "กลุ่มที่ 1"
	reSec := regexp.MustCompile(`กลุ่มที่\s*(\d+)`)
	if match := reSec.FindStringSubmatch(text); len(match) > 1 {
		section = strings.TrimSpace(match[1])
	}

	// Parse Room
	// e.g. "ห้องสอบ CP9421"
	reRoom := regexp.MustCompile(`ห้องสอบ\s+([A-Za-z0-9\-]+)`)
	if match := reRoom.FindStringSubmatch(text); len(match) > 1 {
		room = strings.TrimSpace(match[1])
	}

	// Parse Date & Time
	// e.g. "วันเวลาสอบ 22 มี.ค 69 (17:00 – 19:00 น.)"
	reDateTime := regexp.MustCompile(`วันเวลาสอบ\s+(\d+)\s+([^\s\d]+)\s+(\d+)\s*\(([^)]+)\)`)
	if match := reDateTime.FindStringSubmatch(text); len(match) > 4 {
		day := match[1]
		monthTh := match[2]
		yearTh := match[3]
		rawTime := match[4]

		dateStr = formatThaiDate(day, monthTh, yearTh)
		timeStr = formatTime(rawTime)
	}

	// Determine labels
	var labels []string
	if len(defaultLabels) > 0 {
		labels = append(labels, defaultLabels...)
	}

	if strings.Contains(text, "ปฏิบัติ") || strings.Contains(strings.ToLower(text), "lab") {
		hasLab := false
		for _, l := range labels {
			if strings.EqualFold(l, "LAB") {
				hasLab = true
				break
			}
		}
		if !hasLab {
			labels = append([]string{"LAB"}, labels...)
		}
	}

	// Extract Student Rows
	// Matches: StudentID (e.g. 683380010-2), Branch (e.g. CP-Cy), Seat (e.g. CP9421-67)
	// Example lines in pdftotext output:
	// Student ID list block: "683380010-2"
	// Branch list block: "CP-Cy"
	// Seat list block: "CP9421-67"

	studentIDs := regexp.MustCompile(`\b(\d{9}-\d)\b`).FindAllString(text, -1)
	branches := regexp.MustCompile(`\b(CP-[A-Za-z0-9]+|[A-Z]{2,4})\b`).FindAllString(text, -1)
	// Filter seats to those appearing in seat column (e.g. CP9421-67 or room-numbered seats)
	var seatsCol []string
	if room != "" {
		reRoomSeat := regexp.MustCompile(fmt.Sprintf(`\b%s-\d+\b`, regexp.QuoteMeta(room)))
		seatsCol = reRoomSeat.FindAllString(text, -1)
	}

	// Fallback if room prefix is not present in seat numbers
	if len(seatsCol) < len(studentIDs) {
		seatsCol = nil
		seatMatches := regexp.MustCompile(`\b([A-Z0-9]+-\d+)\b`).FindAllString(text, -1)
		for _, sm := range seatMatches {
			if !strings.Contains(sm, "6833800") && sm != "2-2" && sm != "2-2-5" {
				seatsCol = append(seatsCol, sm)
			}
		}
	}

	var extractedSeats []models.Seat
	count := len(studentIDs)
	if count == 0 {
		return nil, fmt.Errorf("no student IDs found in PDF roster")
	}

	for i := 0; i < count; i++ {
		stID := models.NormalizeStudentID(studentIDs[i])
		if stID == "" {
			continue
		}

		branchVal := ""
		if i < len(branches) {
			branchVal = branches[i]
		}

		seatVal := ""
		if i < len(seatsCol) {
			seatVal = seatsCol[i]
		}

		extractedSeats = append(extractedSeats, models.Seat{
			Sheet:       "PDF_Import",
			Date:        dateStr,
			Room:        room,
			Subject:     subject,
			SubjectName: subjectName,
			Section:     section,
			StudentID:   stID,
			Time:        timeStr,
			Seat:        seatVal,
			Note:        "",
			ExamRound:   defaultRound,
			Branch:      branchVal,
			Labels:      labels,
			RoomLayout:  roomLayout,
			CustomID:    customID,
		})
	}

	return extractedSeats, nil
}

func formatThaiDate(day, rawMonth, yearTh string) string {
	cleanMonth := strings.ReplaceAll(rawMonth, ".", "")
	monthMap := map[string]string{
		"มค": "01", "กพ": "02", "มีค": "03", "เมย": "04",
		"พค": "05", "มิย": "06", "กค": "07", "สค": "08",
		"กย": "09", "ตค": "10", "พย": "11", "ธค": "12",
	}
	month, ok := monthMap[cleanMonth]
	if !ok {
		return ""
	}
	if len(day) == 1 {
		day = "0" + day
	}
	yInt, _ := strconv.Atoi(yearTh)
	yearAD := (2500 + yInt) - 543
	return fmt.Sprintf("%d-%s-%s", yearAD, month, day)
}

func formatTime(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, " น.", "")
	raw = strings.ReplaceAll(raw, "น.", "")
	raw = strings.ReplaceAll(raw, " – ", " - ")
	raw = strings.ReplaceAll(raw, " - ", "-")
	raw = strings.ReplaceAll(raw, ":", ".")
	parts := strings.Split(raw, "-")
	if len(parts) == 2 {
		return fmt.Sprintf("%s - %s", strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}
	return raw
}

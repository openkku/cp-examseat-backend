// Package xlsx imports exam seats from the faculty's Excel seating workbooks.
package xlsx

import (
	"fmt"
	"log"
	"strings"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"

	"github.com/xuri/excelize/v2"
)

// ExtractSeats parses an Excel file and returns extracted seats
func ExtractSeats(filePath string, roundID string, labels []string, roomLayout string, customID string) ([]domain.Seat, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer f.Close()

	var seats []domain.Seat

	sheets := f.GetSheetList()
	for _, sheetName := range sheets {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Printf("⚠️ Warning: Could not read rows in sheet %q: %v", sheetName, err)
			continue
		}

		rowPtr := 0
		for rowPtr < len(rows) {
			startRow := -1
			for r := rowPtr; r < len(rows); r++ {
				visible, err := f.GetRowVisible(sheetName, r+1)
				if err == nil && !visible {
					continue
				}
				found := false
				for _, colVal := range rows[r] {
					if strings.Contains(colVal, "ใบรายชื่อ") {
						startRow = r
						found = true
						break
					}
				}
				if found {
					break
				}
			}

			if startRow == -1 {
				break
			}

			headerRow := -1
			var room, subject, subjectName, section, timeVal string

			limit := startRow + 10
			if limit > len(rows) {
				limit = len(rows)
			}

			for r := startRow; r < limit; r++ {
				visible, err := f.GetRowVisible(sheetName, r+1)
				if err == nil && !visible {
					continue
				}
				rowCells := rows[r]
				if len(rowCells) == 0 {
					continue
				}
				valA := rowCells[0]
				var valB string
				if len(rowCells) > 1 {
					valB = rowCells[1]
				}

				if strings.Contains(valA, "รายวิชา") {
					subjectRaw := strings.TrimSpace(valB)
					subjectRaw = strings.ReplaceAll(subjectRaw, ": ", "")
					subjectRaw = strings.TrimSpace(subjectRaw)

					nbspIdx := strings.Index(subjectRaw, "\u00a0")
					spaceIdx := strings.Index(subjectRaw, " ")

					if nbspIdx != -1 {
						subject = strings.TrimSpace(subjectRaw[:nbspIdx])
						subjectName = strings.TrimSpace(subjectRaw[nbspIdx+len("\u00a0"):])
					} else if spaceIdx != -1 {
						subject = strings.TrimSpace(subjectRaw[:spaceIdx])
						subjectName = strings.TrimSpace(subjectRaw[spaceIdx+1:])
					} else {
						subject = subjectRaw
						subjectName = ""
					}

					var sectionRaw string
					if len(rowCells) > 5 {
						sectionRaw = rowCells[5]
					}
					section = strings.TrimSpace(strings.ReplaceAll(sectionRaw, "SEC.", ""))
				}

				if strings.Contains(valA, "เวลาสอบ") {
					timeRaw := strings.TrimSpace(valB)
					timeVal = strings.TrimSpace(strings.ReplaceAll(timeRaw, " น.", ""))
				}

				if strings.Contains(valA, "ห้องสอบ") {
					room = strings.TrimSpace(valB)
				}

				if strings.Contains(valB, "รหัส") {
					headerRow = r
					break
				}
			}

			if headerRow == -1 {
				rowPtr = startRow + 1
				continue
			}

			dataRow := headerRow + 1

			for dataRow < len(rows) {
				visible, err := f.GetRowVisible(sheetName, dataRow+1)
				if err == nil && !visible {
					dataRow++
					continue
				}

				rowCells := rows[dataRow]
				var studentID, seat, note, branch string

				if len(rowCells) > 1 {
					studentID = strings.TrimSpace(rowCells[1])
				}
				if len(rowCells) > 3 {
					branch = strings.TrimSpace(rowCells[3])
				}
				if len(rowCells) > 4 {
					seat = strings.TrimSpace(rowCells[4])
				}
				if len(rowCells) > 5 {
					note = strings.TrimSpace(rowCells[5])
				}

				if studentID == "" && seat == "" {
					break
				}

				cleanID := domain.NormalizeStudentID(studentID)
				if cleanID != "" {
					cleanDate := domain.ParseThaiDate(sheetName)
					seats = append(seats, domain.Seat{
						Sheet:       sheetName,
						Date:        cleanDate,
						Room:        room,
						Subject:     subject,
						SubjectName: subjectName,
						Section:     section,
						StudentID:   cleanID,
						Time:        timeVal,
						Seat:        seat,
						Note:        note,
						ExamRound:   roundID,
						Branch:      branch,
						Labels:      labels,
						RoomLayout:  roomLayout,
						CustomID:    customID,
					})
				}

				dataRow++
			}

			rowPtr = dataRow
		}
	}

	return seats, nil
}

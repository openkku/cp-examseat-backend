package database

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// SeatSelect selects every column scanned by ScanSeats.
const SeatSelect = `
        SELECT
            es.sheet, es.date, es.time_start, COALESCE(es.time_end, ''), es.room, es.subject_id, es.section, st_seat.student_id, st_seat.seat, es.note,
            COALESCE(sub.name, '') as subject_name, es.exam_round, COALESCE(st.branch, '') as branch,
            COALESCE((SELECT GROUP_CONCAT(label, ',') FROM session_labels WHERE session_id = es.id), '') as labels,
            COALESCE(es.room_layout, '') as room_layout, COALESCE(es.custom_id, '') as custom_id, COALESCE(es.category, '') as category
        FROM exam_seats st_seat
        JOIN exam_sessions es ON st_seat.session_id = es.id
        JOIN students st ON st_seat.student_id = st.id
        LEFT JOIN subjects sub ON es.subject_id = sub.id AND es.exam_round = sub.exam_round
`

// ScanSeats reads rows selected with SeatSelect.
func ScanSeats(rows *sql.Rows) ([]domain.Seat, error) {
	var seats []domain.Seat
	for rows.Next() {
		var s domain.Seat
		var rawLabels string
		err := rows.Scan(
			&s.Sheet,
			&s.Date,
			&s.TimeStart,
			&s.TimeEnd,
			&s.Room,
			&s.Subject,
			&s.Section,
			&s.StudentID,
			&s.Seat,
			&s.Note,
			&s.SubjectName,
			&s.ExamRound,
			&s.Branch,
			&rawLabels,
			&s.RoomLayout,
			&s.CustomID,
			&s.Category,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning seats: %w", err)
		}
		s.Time = s.GetTime()
		s.Labels = domain.ParseLabels(rawLabels)
		seats = append(seats, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over seats rows: %w", err)
	}

	return seats, nil
}

// Rounds lists every round, newest first.
func Rounds(ctx context.Context, db *sql.DB) ([]domain.Round, error) {
	query := `
		SELECT id, label 
		FROM round_info 
		ORDER BY rowid DESC
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rounds: %w", err)
	}
	defer rows.Close()

	var rounds []domain.Round
	for rows.Next() {
		var ro domain.Round
		if err := rows.Scan(&ro.ID, &ro.Label); err != nil {
			return nil, fmt.Errorf("failed to scan round: %w", err)
		}
		rounds = append(rounds, ro)
	}

	sort.Slice(rounds, func(i, j int) bool {
		return domain.CompareRounds(rounds[i].ID, rounds[j].ID)
	})

	return rounds, nil
}

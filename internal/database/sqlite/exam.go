package sqlite

import (
	"context"
	"fmt"

	"github.com/openkku/cp-examseat-backend/internal/exam"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// GetSeatsByID returns every seat of one student ordered by date and start time.
func (d *DB) GetSeatsByID(ctx context.Context, opts domain.SeatQuery) ([]domain.Seat, error) {
	query := seatSelect + " WHERE st_seat.student_id = ?"
	args := []any{opts.StudentID}

	if opts.Round != nil {
		query += " AND es.exam_round = ?"
		args = append(args, *opts.Round)
	}

	if opts.Sheet != nil {
		query += " AND es.sheet = ?"
		args = append(args, *opts.Sheet)
	}

	query += " ORDER BY es.date ASC, es.time_start ASC"

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying seats: %w", err)
	}
	defer rows.Close()

	return scanSeats(rows)
}

// GetSeats returns the roster of one room and exam slot ordered by seat.
func (d *DB) GetSeats(ctx context.Context, opts exam.ExploreQuery) ([]domain.Seat, error) {
	query := seatSelect + " WHERE es.exam_round = ? AND es.room = ? AND (es.date = ? OR ? = '')"
	args := []any{opts.Round, opts.Room, opts.Date, opts.Date}

	if opts.Time != "" {
		query += " AND (es.time_start = ? OR (es.time_start || '-' || COALESCE(es.time_end, '')) = ?)"
		args = append(args, opts.Time, opts.Time)
	}

	if opts.Seat != nil && *opts.Seat != "" {
		query += " AND st_seat.seat = ?"
		args = append(args, *opts.Seat)
	}

	query += " ORDER BY st_seat.seat ASC"

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying seats: %w", err)
	}
	defer rows.Close()

	return scanSeats(rows)
}

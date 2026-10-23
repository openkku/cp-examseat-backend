// Package database stores checkex lookups and the seats they fetched in the SQL database.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/checkex"
	platformdb "github.com/openkku/cp-examseat-backend/internal/platform/database"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Repo implements checkex.Repository.
type Repo struct {
	db *platformdb.DB
}

var _ checkex.Repository = (*Repo)(nil)

// New creates a Repo on db.
func New(db *platformdb.DB) *Repo {
	return &Repo{db: db}
}

// HasRegularSeats reports whether the student has seats in the round outside custom datasets.
func (r *Repo) HasRegularSeats(ctx context.Context, roundID string, studentID string) (bool, error) {
	var n int
	if err := r.db.SQL().QueryRowContext(ctx, regularSeatCount, roundID, studentID).Scan(&n); err != nil {
		return false, fmt.Errorf("failed to check existing seats: %w", err)
	}
	return n > 0, nil
}

// regularSeatCount counts a student's seats in a round, not counting custom datasets.
const regularSeatCount = `
		SELECT COUNT(*) FROM exam_seats st JOIN exam_sessions es ON st.session_id = es.id
		WHERE es.exam_round = ? AND st.student_id = ? AND COALESCE(es.custom_id, '') = ''`

// EnsureRound creates the round if it does not exist yet, keeping an existing label.
func (r *Repo) EnsureRound(ctx context.Context, roundID string, displayName string) error {
	if _, err := r.db.SQL().ExecContext(ctx, "INSERT OR IGNORE INTO round_info (id, label) VALUES (?, ?)", roundID, displayName); err != nil {
		return fmt.Errorf("failed to ensure round: %w", err)
	}
	return nil
}

// AddStudentSeats appends one student's seats to an existing round. Unlike AddRound it reuses
// sessions already in the database, so students fetched one at a time still share a session.
// It is a no-op (besides marking the lookup) if the student already has seats in the round,
// not counting custom datasets.
func (r *Repo) AddStudentSeats(ctx context.Context, roundID string, studentID string, seats []domain.Seat) error {
	tx, err := r.db.SQL().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var existing int
	if err := tx.QueryRowContext(ctx, regularSeatCount, roundID, studentID).Scan(&existing); err != nil {
		return fmt.Errorf("failed to check existing seats: %w", err)
	}

	if existing == 0 {
		for _, s := range seats {
			if err := addStudentSeat(ctx, tx, roundID, studentID, s); err != nil {
				return err
			}
		}
	}

	if err := upsertLookup(ctx, tx, roundID, studentID, checkex.LookupFound); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func addStudentSeat(ctx context.Context, tx *sql.Tx, roundID string, studentID string, s domain.Seat) error {
	category := s.ResolveCategory()

	var sessionID int64
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM exam_sessions
		WHERE exam_round = ? AND category = ? AND custom_id IS ? AND sheet = ? AND date = ? AND time_start = ?
		  AND time_end IS ? AND room = ? AND subject_id = ? AND section = ? AND note IS ? AND room_layout IS ?
		LIMIT 1`,
		roundID, category, s.CustomID, s.Sheet, s.Date, s.TimeStart, s.TimeEnd, s.Room, s.Subject, s.Section, s.Note, s.RoomLayout,
	).Scan(&sessionID)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		res, err := tx.ExecContext(ctx, `
			INSERT INTO exam_sessions (exam_round, category, custom_id, sheet, date, time_start, time_end, room, subject_id, section, note, room_layout)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			roundID, category, s.CustomID, s.Sheet, s.Date, s.TimeStart, s.TimeEnd, s.Room, s.Subject, s.Section, s.Note, s.RoomLayout)
		if err != nil {
			return fmt.Errorf("failed to insert exam session: %w", err)
		}
		if sessionID, err = res.LastInsertId(); err != nil {
			return fmt.Errorf("failed to get last insert id for session: %w", err)
		}
		for _, lbl := range s.Labels {
			if lbl = strings.TrimSpace(lbl); lbl != "" {
				if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO session_labels (session_id, label) VALUES (?, ?)", sessionID, lbl); err != nil {
					return fmt.Errorf("failed to insert label: %w", err)
				}
			}
		}
	case err != nil:
		return fmt.Errorf("failed to look up exam session: %w", err)
	}

	if s.Subject != "" {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO subjects (id, exam_round, name) VALUES (?, ?, ?)
			ON CONFLICT(id, exam_round) DO UPDATE SET name = CASE WHEN excluded.name != '' THEN excluded.name ELSE name END`,
			s.Subject, roundID, s.SubjectName); err != nil {
			return fmt.Errorf("failed to insert subject: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO students (id, branch) VALUES (?, ?)", studentID, s.Branch); err != nil {
		return fmt.Errorf("failed to insert student: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO exam_seats (session_id, student_id, seat) VALUES (?, ?, ?)", sessionID, studentID, s.Seat); err != nil {
		return fmt.Errorf("failed to insert exam seat: %w", err)
	}
	return nil
}

// GetLookup returns the recorded upstream lookup, or nil when there is none.
func (r *Repo) GetLookup(ctx context.Context, roundID string, studentID string) (*checkex.Lookup, error) {
	var status string
	var fetchedAt int64
	err := r.db.SQL().QueryRowContext(ctx, "SELECT status, fetched_at FROM upstream_lookups WHERE exam_round = ? AND student_id = ?", roundID, studentID).Scan(&status, &fetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query upstream lookup: %w", err)
	}
	return &checkex.Lookup{Status: status, FetchedAt: time.Unix(fetchedAt, 0)}, nil
}

// MarkNotFound records that checkex has no seats for the student in the round.
func (r *Repo) MarkNotFound(ctx context.Context, roundID string, studentID string) error {
	tx, err := r.db.SQL().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	if err := upsertLookup(ctx, tx, roundID, studentID, checkex.LookupNotFound); err != nil {
		return err
	}
	return tx.Commit()
}

func upsertLookup(ctx context.Context, tx *sql.Tx, roundID string, studentID string, status string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO upstream_lookups (exam_round, student_id, status, fetched_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(exam_round, student_id) DO UPDATE SET status = excluded.status, fetched_at = excluded.fetched_at`,
		roundID, studentID, status, time.Now().Unix()); err != nil {
		return fmt.Errorf("failed to record upstream lookup: %w", err)
	}
	return nil
}

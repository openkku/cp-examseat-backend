package sqlite

import (
	"context"
	"fmt"
	"sort"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// RoundExists reports whether the round is stored.
func (d *DB) RoundExists(ctx context.Context, roundID string) (bool, error) {
	var n int
	if err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM round_info WHERE id = ?", roundID).Scan(&n); err != nil {
		return false, fmt.Errorf("failed to look up round: %w", err)
	}
	return n > 0, nil
}

// RoundSummaries lists every round with seat and student counts, newest first.
func (d *DB) RoundSummaries(ctx context.Context) ([]admin.RoundSummary, error) {
	rounds, err := d.rounds(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*admin.RoundSummary, len(rounds))
	out := make([]admin.RoundSummary, len(rounds))
	for i, ro := range rounds {
		out[i] = admin.RoundSummary{ID: ro.ID, Label: ro.Label, CustomDatasets: []admin.CustomDataset{}}
		byID[ro.ID] = &out[i]
	}

	rows, err := d.db.QueryContext(ctx, `
		SELECT es.exam_round, COALESCE(es.custom_id, ''), COUNT(*)
		FROM exam_seats st
		JOIN exam_sessions es ON st.session_id = es.id
		GROUP BY es.exam_round, COALESCE(es.custom_id, '')`)
	if err != nil {
		return nil, fmt.Errorf("failed to count seats: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var round, customID string
		var count int
		if err := rows.Scan(&round, &customID, &count); err != nil {
			return nil, err
		}
		s, ok := byID[round]
		if !ok {
			continue
		}
		s.Seats += count
		if customID == "" {
			s.InScheduleSeats += count
		} else {
			s.CustomDatasets = append(s.CustomDatasets, admin.CustomDataset{ID: customID, Seats: count})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	students, err := d.db.QueryContext(ctx, `
		SELECT es.exam_round, COUNT(DISTINCT st.student_id)
		FROM exam_seats st
		JOIN exam_sessions es ON st.session_id = es.id
		GROUP BY es.exam_round`)
	if err != nil {
		return nil, fmt.Errorf("failed to count students: %w", err)
	}
	defer students.Close()
	for students.Next() {
		var round string
		var count int
		if err := students.Scan(&round, &count); err != nil {
			return nil, err
		}
		if s, ok := byID[round]; ok {
			s.Students = count
		}
	}

	for i := range out {
		sort.Slice(out[i].CustomDatasets, func(a, b int) bool { return out[i].CustomDatasets[a].ID < out[i].CustomDatasets[b].ID })
	}
	return out, students.Err()
}

// DeleteRound removes a round with all its sessions (in-schedule and custom),
// seats, labels, subjects and recorded checkex lookups.
func (d *DB) DeleteRound(ctx context.Context, roundID string) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "DELETE FROM round_info WHERE id = ?", roundID)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrRoundNotFound
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM upstream_lookups WHERE exam_round = ?", roundID); err != nil {
		return fmt.Errorf("failed to clear upstream lookups: %w", err)
	}
	return tx.Commit()
}

// SetRoundLabel renames a round.
func (d *DB) SetRoundLabel(ctx context.Context, roundID string, label string) error {
	res, err := d.db.ExecContext(ctx, "UPDATE round_info SET label = ? WHERE id = ?", label, roundID)
	if err != nil {
		return fmt.Errorf("failed to rename round: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrRoundNotFound
	}
	return nil
}

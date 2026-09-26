package sqlite

import (
	"context"
	"fmt"
	"sort"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// RoundSummaries lists every round with seat and student counts, newest first.
func (d *Database) RoundSummaries(ctx context.Context) ([]models.RoundSummary, error) {
	rounds, err := d.GetRounds(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*models.RoundSummary, len(rounds))
	out := make([]models.RoundSummary, len(rounds))
	for i, r := range rounds {
		out[i] = models.RoundSummary{ID: r.ID, Label: r.Label, CustomDatasets: []models.CustomDataset{}}
		byID[r.ID] = &out[i]
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
			s.CustomDatasets = append(s.CustomDatasets, models.CustomDataset{ID: customID, Seats: count})
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
// seats, labels and subjects.
func (d *Database) DeleteRound(ctx context.Context, roundID string) error {
	res, err := d.db.ExecContext(ctx, "DELETE FROM round_info WHERE id = ?", roundID)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return models.ErrRoundNotFound
	}
	return nil
}

// SetRoundLabel renames a round.
func (d *Database) SetRoundLabel(ctx context.Context, roundID string, label string) error {
	res, err := d.db.ExecContext(ctx, "UPDATE round_info SET label = ? WHERE id = ?", label, roundID)
	if err != nil {
		return fmt.Errorf("failed to rename round: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return models.ErrRoundNotFound
	}
	return nil
}

// Backup writes a consistent snapshot with VACUUM INTO, which is safe while
// the server keeps serving reads and writes.
func (d *Database) Backup(ctx context.Context, destPath string) error {
	if _, err := d.db.ExecContext(ctx, "VACUUM INTO ?", destPath); err != nil {
		return fmt.Errorf("failed to back up database: %w", err)
	}
	return nil
}

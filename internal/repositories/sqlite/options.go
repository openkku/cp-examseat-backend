package sqlite

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

// GetOptions lists the distinct dates, times or rooms of a round for the explorer filters.
func (d *Database) GetOptions(ctx context.Context, opts models.OptionsQuery) ([]string, error) {
	switch opts.Mode {
	case models.OptionDates:
		query := "SELECT DISTINCT date, room FROM exam_sessions WHERE exam_round = ? ORDER BY date"
		rows, err := d.db.QueryContext(ctx, query, opts.Round)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		dateSet := make(map[string]bool)
		var result []string
		for rows.Next() {
			var dateVal, roomVal string
			if err := rows.Scan(&dateVal, &roomVal); err != nil {
				return nil, err
			}
			if !opts.AcceptsRoom(roomVal) {
				continue
			}
			if !dateSet[dateVal] {
				dateSet[dateVal] = true
				result = append(result, dateVal)
			}
		}
		return result, nil

	case models.OptionTimes:
		query := "SELECT DISTINCT time_start, time_end, room FROM exam_sessions WHERE exam_round = ? AND date = ? ORDER BY time_start"
		rows, err := d.db.QueryContext(ctx, query, opts.Round, opts.Date)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		timeSet := make(map[string]bool)
		var result []string
		for rows.Next() {
			var timeStartVal, timeEndVal, roomVal string
			if err := rows.Scan(&timeStartVal, &timeEndVal, &roomVal); err != nil {
				return nil, err
			}
			if !opts.AcceptsRoom(roomVal) {
				continue
			}
			timeStr := timeStartVal
			if timeEndVal != "" {
				timeStr = timeStartVal + "-" + timeEndVal
			}
			if !timeSet[timeStr] {
				timeSet[timeStr] = true
				result = append(result, timeStr)
			}
		}
		return result, nil

	case models.OptionRooms:
		query := "SELECT DISTINCT room FROM exam_sessions WHERE exam_round = ? AND (date = ? OR ? = '') AND (time_start = ? OR (time_start || '-' || COALESCE(time_end, '')) = ? OR ? = '') ORDER BY room"
		rows, err := d.db.QueryContext(ctx, query, opts.Round, opts.Date, opts.Date, opts.Time, opts.Time, opts.Time)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var result []string
		for rows.Next() {
			var roomVal string
			if err := rows.Scan(&roomVal); err != nil {
				return nil, err
			}
			if !opts.AcceptsRoom(roomVal) {
				continue
			}
			result = append(result, roomVal)
		}
		return result, nil

	default:
		return nil, models.ErrInvalidOptionMode
	}
}

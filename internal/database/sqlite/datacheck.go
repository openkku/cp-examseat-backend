package sqlite

import (
	"context"

	"github.com/openkku/cp-examseat-backend/internal/datacheck"
)

// RoomSeats returns distinct seat identifiers grouped by exam round and room.
func (d *DB) RoomSeats(ctx context.Context) ([]datacheck.RoomSeats, error) {
	query := `
		SELECT es.exam_round, es.room, st.seat
		FROM exam_seats st
		JOIN exam_sessions es ON st.session_id = es.id
		GROUP BY es.exam_round, es.room, st.seat
		ORDER BY es.exam_round, es.room, st.seat
	`
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type key struct {
		round string
		room  string
	}
	var keys []key
	seatMap := make(map[key][]string)

	for rows.Next() {
		var round, room, seat string
		if err := rows.Scan(&round, &room, &seat); err != nil {
			return nil, err
		}
		k := key{round: round, room: room}
		if _, exists := seatMap[k]; !exists {
			keys = append(keys, k)
		}
		seatMap[k] = append(seatMap[k], seat)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]datacheck.RoomSeats, 0, len(keys))
	for _, k := range keys {
		result = append(result, datacheck.RoomSeats{
			Round: k.round,
			Room:  k.room,
			Seats: seatMap[k],
		})
	}
	return result, nil
}

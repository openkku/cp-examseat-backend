package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/database/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

func TestRoomSeats(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "exams.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Add seats in round 1
	seatsR1 := []domain.Seat{
		{Room: "SC1101", Seat: "A1", Date: "2026-10-01", Time: "09:00-12:00", StudentID: "1"},
		{Room: "SC1101", Seat: "A2", Date: "2026-10-01", Time: "09:00-12:00", StudentID: "2"},
		{Room: "SC1102", Seat: "B1", Date: "2026-10-01", Time: "09:00-12:00", StudentID: "3"},
	}
	if err := db.AddRound(ctx, "round_1", "Round 1", seatsR1); err != nil {
		t.Fatal(err)
	}

	// Add seats in round 2
	seatsR2 := []domain.Seat{
		{Room: "SC1101", Seat: "A1", Date: "2026-10-02", Time: "13:00-16:00", StudentID: "4"},
	}
	if err := db.AddRound(ctx, "round_2", "Round 2", seatsR2); err != nil {
		t.Fatal(err)
	}

	usage, err := db.RoomSeats(ctx)
	if err != nil {
		t.Fatalf("RoomSeats() error: %v", err)
	}

	if len(usage) != 3 {
		t.Fatalf("expected 3 (round, room) groups, got %d: %+v", len(usage), usage)
	}

	// Group 1: round_1, SC1101 -> A1, A2
	if usage[0].Round != "round_1" || usage[0].Room != "SC1101" || len(usage[0].Seats) != 2 {
		t.Errorf("group 0 mismatch: %+v", usage[0])
	}
	// Group 2: round_1, SC1102 -> B1
	if usage[1].Round != "round_1" || usage[1].Room != "SC1102" || len(usage[1].Seats) != 1 {
		t.Errorf("group 1 mismatch: %+v", usage[1])
	}
	// Group 3: round_2, SC1101 -> A1
	if usage[2].Round != "round_2" || usage[2].Room != "SC1101" || len(usage[2].Seats) != 1 {
		t.Errorf("group 2 mismatch: %+v", usage[2])
	}
}

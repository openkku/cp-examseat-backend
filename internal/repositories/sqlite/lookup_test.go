package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/repositories/sqlite"
)

func TestAddStudentSeatsSharesSessions(t *testing.T) {
	db, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	round := "final_1_2569"
	if err := db.EnsureRound(ctx, round, "ปลายภาค 1/2569"); err != nil {
		t.Fatal(err)
	}

	seat := func(id, seat string) []models.Seat {
		return []models.Seat{{
			Sheet: "20 ต.ค. 69 (เช้า)", Date: "2026-10-20", TimeStart: "08.30", TimeEnd: "11.30",
			Category: "IN_SCHEDULE", Room: "SC.7401", Subject: "CP422022", SubjectName: "Database",
			Section: "02", Note: "ปกติ", StudentID: id, Seat: seat, Branch: "CP-Cy", ExamRound: round,
		}}
	}

	if l, _ := db.GetLookup(ctx, round, "6833800001"); l != nil {
		t.Fatalf("expected no lookup marker, got %+v", l)
	}
	if err := db.AddStudentSeats(ctx, round, "6833800001", seat("6833800001", "B1")); err != nil {
		t.Fatal(err)
	}
	if err := db.AddStudentSeats(ctx, round, "6833800002", seat("6833800002", "B2")); err != nil {
		t.Fatal(err)
	}
	// A repeated insert for the same student must not duplicate seats.
	if err := db.AddStudentSeats(ctx, round, "6833800001", seat("6833800001", "B1")); err != nil {
		t.Fatal(err)
	}

	got, err := db.GetSeats(ctx, models.ExploreQuery{Round: round, Room: "SC.7401", Date: "2026-10-20", Time: "08.30-11.30"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 seats in shared session, got %d: %+v", len(got), got)
	}

	mine, err := db.GetSeatsByID(ctx, models.SeatQuery{StudentID: "6833800001", Round: &round})
	if err != nil || len(mine) != 1 || mine[0].Seat != "B1" {
		t.Fatalf("unexpected seats for student: %+v, %v", mine, err)
	}

	if l, _ := db.GetLookup(ctx, round, "6833800001"); l == nil || l.Status != models.LookupFound {
		t.Fatalf("expected found marker, got %+v", l)
	}

	if err := db.MarkNotFound(ctx, round, "6933800009"); err != nil {
		t.Fatal(err)
	}
	if l, _ := db.GetLookup(ctx, round, "6933800009"); l == nil || l.Status != models.LookupNotFound {
		t.Fatalf("expected not_found marker, got %+v", l)
	}

	if err := db.PurgeRound(ctx, round); err != nil {
		t.Fatal(err)
	}
	if l, _ := db.GetLookup(ctx, round, "6833800001"); l != nil {
		t.Fatalf("expected markers purged with round, got %+v", l)
	}
}

func TestAddStudentSeatsIgnoresCustomDatasets(t *testing.T) {
	db, err := sqlite.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	round := "final_1_2569"
	id := "6833800001"
	custom := []models.Seat{{
		Sheet: "นัดสอบปลายภาค (บรรยาย)", Date: "2026-10-17", TimeStart: "13.00", TimeEnd: "16.00",
		CustomID: "FINAL_1_2569_CP421025_LEC", Room: "CP9226", Subject: "CP421025", Section: "1",
		StudentID: id, Seat: "CP9226-1", Branch: "CP-Cy", ExamRound: round,
	}}
	if err := db.AddRound(ctx, round, "ปลายภาค 1/2569", custom); err != nil {
		t.Fatal(err)
	}

	upstream := []models.Seat{{
		Sheet: "20 ต.ค. 69 (เช้า)", Date: "2026-10-20", TimeStart: "08.30", TimeEnd: "11.30",
		Category: "IN_SCHEDULE", Room: "SC.7401", Subject: "CP422022", Section: "02",
		StudentID: id, Seat: "B1", Branch: "CP-Cy", ExamRound: round,
	}}
	if err := db.AddStudentSeats(ctx, round, id, upstream); err != nil {
		t.Fatal(err)
	}

	mine, err := db.GetSeatsByID(ctx, models.SeatQuery{StudentID: id, Round: &round})
	if err != nil || len(mine) != 2 {
		t.Fatalf("expected custom and upstream seats, got %+v, %v", mine, err)
	}
}

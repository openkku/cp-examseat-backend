package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/repositories/sqlite"
)

func seedAdmin(t *testing.T) (*sqlite.Database, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.New(filepath.Join(dir, "admin.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	ctx := context.Background()
	main := []models.Seat{
		{StudentID: "1", Date: "2026-09-01", Time: "09.00-12.00", Room: "A", Subject: "CP1"},
		{StudentID: "2", Date: "2026-09-01", Time: "09.00-12.00", Room: "A", Subject: "CP1"},
		{StudentID: "1", Date: "2026-09-02", Time: "09.00-12.00", Room: "B", Subject: "CP2"},
	}
	lab := []models.Seat{{StudentID: "3", Date: "2026-09-03", Time: "13.00", Room: "L", Subject: "CP3", CustomID: "LAB"}}
	if err := db.AddRound(ctx, "mid_1_2569", "กลางภาค 1/2569", main); err != nil {
		t.Fatal(err)
	}
	if err := db.AddRound(ctx, "mid_1_2569", "กลางภาค 1/2569", lab); err != nil {
		t.Fatal(err)
	}
	if err := db.AddRound(ctx, "final_2_2568", "ปลายภาค 2/2568", main[:1]); err != nil {
		t.Fatal(err)
	}
	return db, dir
}

func TestRoundSummaries(t *testing.T) {
	db, _ := seedAdmin(t)
	got, err := db.RoundSummaries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "mid_1_2569" {
		t.Fatalf("summaries = %+v", got)
	}
	mid := got[0]
	if mid.Seats != 4 || mid.InScheduleSeats != 3 || mid.Students != 3 {
		t.Errorf("mid counts = %+v", mid)
	}
	if len(mid.CustomDatasets) != 1 || mid.CustomDatasets[0] != (models.CustomDataset{ID: "LAB", Seats: 1}) {
		t.Errorf("custom datasets = %+v", mid.CustomDatasets)
	}
	if got[1].Seats != 1 || got[1].CustomDatasets == nil {
		t.Errorf("final = %+v (custom datasets must be [] not null)", got[1])
	}
}

func TestDeleteRoundAndRename(t *testing.T) {
	db, _ := seedAdmin(t)
	ctx := context.Background()

	if err := db.SetRoundLabel(ctx, "final_2_2568", "Final"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetRoundLabel(ctx, "nope", "x"); !errors.Is(err, models.ErrRoundNotFound) {
		t.Errorf("rename unknown = %v", err)
	}

	if err := db.DeleteRound(ctx, "mid_1_2569"); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteRound(ctx, "mid_1_2569"); !errors.Is(err, models.ErrRoundNotFound) {
		t.Errorf("second delete = %v", err)
	}

	rounds, _ := db.GetRounds(ctx)
	if len(rounds) != 1 || rounds[0].Label != "Final" {
		t.Errorf("rounds after delete = %+v", rounds)
	}
	var sessions, seats int
	db.RawQueryRow(ctx, "SELECT COUNT(*) FROM exam_sessions WHERE exam_round = 'mid_1_2569'").Scan(&sessions)
	db.RawQueryRow(ctx, "SELECT COUNT(*) FROM exam_seats").Scan(&seats)
	if sessions != 0 || seats != 1 {
		t.Errorf("delete left sessions=%d seats=%d (want 0 and 1)", sessions, seats)
	}
}

func TestBackup(t *testing.T) {
	db, dir := seedAdmin(t)
	dest := filepath.Join(dir, "backup.db")
	if err := db.Backup(context.Background(), dest); err != nil {
		t.Fatal(err)
	}
	if err := db.Backup(context.Background(), dest); err == nil {
		t.Error("backup must refuse to overwrite an existing file")
	}

	restored, err := sqlite.New(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	seats, err := restored.GetAllSeats(context.Background())
	if err != nil || len(seats) != 5 {
		t.Errorf("restored backup has %d seats (%v); want 5", len(seats), err)
	}
}

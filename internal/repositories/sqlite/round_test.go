package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/repositories/sqlite"
)

func TestMultiLabelAndCustomIDPurge(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create sqlite DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	roundID := "2_2568"

	// 1. Add Main Seats
	mainSeats := []models.Seat{
		{
			ExamRound:   roundID,
			StudentID:   "683380010-2",
			Subject:     "CP1000",
			SubjectName: "Computer Intro",
			Date:        "2026-03-20",
			Time:        "09.00 - 12.00",
			Room:        "SC101",
			Seat:        "A01",
			CustomID:    "MAIN_2_2568",
			Labels:      nil,
		},
	}
	if err := db.AddRound(ctx, roundID, "Final 2/2568", mainSeats); err != nil {
		t.Fatalf("Failed to add main seats: %v", err)
	}

	// 2. Add Lab Seats with Custom ID
	labSeats := []models.Seat{
		{
			ExamRound:   roundID,
			StudentID:   "683380010-2",
			Subject:     "CP421024",
			SubjectName: "OOP",
			Date:        "2026-03-22",
			Time:        "17.00 - 19.00",
			Room:        "CP9421",
			Seat:        "CP9421-67",
			CustomID:    "FINAL_2_OOP_LAB_2026",
			Labels:      []string{"LAB", "Lab"},
			RoomLayout:  "CP9421_LAB",
		},
	}
	if err := db.AddRound(ctx, roundID, "Final 2/2568", labSeats); err != nil {
		t.Fatalf("Failed to add lab seats: %v", err)
	}

	// 3. Query student seats and verify both main and lab seats return together
	seats, err := db.GetSeatsByID(ctx, models.SeatQuery{StudentID: "683380010-2", Round: &roundID})
	if err != nil {
		t.Fatalf("Failed to GetSeatsByID: %v", err)
	}

	if len(seats) != 2 {
		t.Fatalf("Expected 2 seats (main + lab), got %d", len(seats))
	}

	// Verify lab seat labels
	var foundLab bool
	for _, s := range seats {
		if s.Subject == "CP421024" {
			foundLab = true
			if len(s.Labels) != 2 || s.Labels[0] != "LAB" || s.Labels[1] != "Lab" {
				t.Errorf("Expected labels ['LAB', 'Lab'], got %v", s.Labels)
			}
			if s.RoomLayout != "CP9421_LAB" {
				t.Errorf("Expected room_layout CP9421_LAB, got %s", s.RoomLayout)
			}
		}
	}
	if !foundLab {
		t.Errorf("Lab seat CP421024 was not found in query results")
	}

	// 4. Test Scoped Purging by custom_id
	if err := db.PurgeCustomDataset(ctx, roundID, "FINAL_2_OOP_LAB_2026"); err != nil {
		t.Fatalf("Failed to PurgeCustomDataset: %v", err)
	}

	// Query again: only main seat should remain
	seatsAfterPurge, err := db.GetSeatsByID(ctx, models.SeatQuery{StudentID: "683380010-2", Round: &roundID})
	if err != nil {
		t.Fatalf("Failed to GetSeatsByID after purge: %v", err)
	}

	if len(seatsAfterPurge) != 1 {
		t.Fatalf("Expected 1 seat after scoped purge, got %d", len(seatsAfterPurge))
	}
	if seatsAfterPurge[0].Subject != "CP1000" {
		t.Errorf("Expected remaining seat subject CP1000, got %s", seatsAfterPurge[0].Subject)
	}
}

func TestGetOptionsCascadeFilter(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_options.db")

	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	roundID := "test_cascade_round"

	// Only CP.9127 has a configured seating layout
	onlyConfigured := func(room string) bool { return room == "CP.9127" }

	seats := []models.Seat{
		{
			ExamRound: roundID,
			StudentID: "11111",
			Date:      "2026-08-23",
			Time:      "13.00-16.00",
			Room:      "ห้อง 9525, 9127", // Unconfigured/invalid room
			Subject:   "CP422022",
		},
		{
			ExamRound: roundID,
			StudentID: "22222",
			Date:      "2026-08-24",
			Time:      "08.30-11.30",
			Room:      "CP.9127", // Valid room
			Subject:   "CP352001",
		},
	}

	if err := db.AddRound(ctx, roundID, "Test Cascade", seats); err != nil {
		t.Fatalf("AddRound failed: %v", err)
	}

	// 1. Test "dates" mode: 2026-08-23 should be filtered out, only 2026-08-24 returned
	dates, err := db.GetOptions(ctx, models.OptionsQuery{Mode: "dates", Round: roundID, RoomFilter: onlyConfigured})
	if err != nil {
		t.Fatalf("GetOptions dates failed: %v", err)
	}
	if len(dates) != 1 || dates[0] != "2026-08-24" {
		t.Errorf("Expected dates ['2026-08-24'], got %v", dates)
	}

	// 2. Test "times" mode for valid date
	times, err := db.GetOptions(ctx, models.OptionsQuery{Mode: "times", Round: roundID, Date: "2026-08-24", RoomFilter: onlyConfigured})
	if err != nil {
		t.Fatalf("GetOptions times failed: %v", err)
	}
	if len(times) != 1 || times[0] != "08.30-11.30" {
		t.Errorf("Expected times ['08.30-11.30'], got %v", times)
	}

	// 3. Test "rooms" mode for valid time slot
	rooms, err := db.GetOptions(ctx, models.OptionsQuery{Mode: "rooms", Round: roundID, Date: "2026-08-24", Time: "08.30-11.30", RoomFilter: onlyConfigured})
	if err != nil {
		t.Fatalf("GetOptions rooms failed: %v", err)
	}
	if len(rooms) != 1 || rooms[0] != "CP.9127" {
		t.Errorf("Expected rooms ['CP.9127'], got %v", rooms)
	}

	// 4. Test "rooms" mode for invalid time slot: should return empty slice
	invalidRooms, err := db.GetOptions(ctx, models.OptionsQuery{Mode: "rooms", Round: roundID, Date: "2026-08-23", Time: "13.00-16.00", RoomFilter: onlyConfigured})
	if err != nil {
		t.Fatalf("GetOptions invalid rooms failed: %v", err)
	}
	if len(invalidRooms) != 0 {
		t.Errorf("Expected empty rooms slice for invalid room slot, got %v", invalidRooms)
	}
}

func TestInVsOutOfScheduleCategory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_cat.db")

	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	roundID := "test_cat_round"

	seats := []models.Seat{
		{
			ExamRound: roundID,
			StudentID: "111111111-1",
			Date:      "2026-09-01",
			Time:      "09.00-12.00",
			Room:      "SC101",
			Subject:   "CP1001",
			Category:  "IN_SCHEDULE",
		},
		{
			ExamRound: roundID,
			StudentID: "222222222-2",
			Date:      "2026-09-02",
			Time:      "13.00-15.00",
			Room:      "CP9421",
			Subject:   "CP422021",
			CustomID:  "MID_1_2569_CP422021_LEC",
			Labels:    []string{"นัดสอบนอกตาราง"},
		},
	}

	if err := db.AddRound(ctx, roundID, "Test Category", seats); err != nil {
		t.Fatalf("AddRound failed: %v", err)
	}

	allSeats, err := db.GetAllSeats(ctx)
	if err != nil {
		t.Fatalf("GetAllSeats failed: %v", err)
	}

	if len(allSeats) != 2 {
		t.Fatalf("Expected 2 seats, got %d", len(allSeats))
	}

	for _, s := range allSeats {
		if s.Subject == "CP1001" && s.Category != "IN_SCHEDULE" {
			t.Errorf("Expected CP1001 category IN_SCHEDULE, got %s", s.Category)
		}
		if s.Subject == "CP422021" && s.Category != "OUT_OF_SCHEDULE" {
			t.Errorf("Expected CP422021 category OUT_OF_SCHEDULE, got %s", s.Category)
		}
	}
}

func TestPurgeRoundPreservesCustomSessions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_purge.db")

	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	roundID := "purge_test_round"

	seats := []models.Seat{
		{
			ExamRound: roundID,
			StudentID: "100000000-1",
			Date:      "2026-09-01",
			Time:      "09.00-12.00",
			Room:      "SC101",
			Subject:   "CP1000",
			Category:  "IN_SCHEDULE",
		},
		{
			ExamRound: roundID,
			StudentID: "200000000-2",
			Date:      "2026-09-02",
			Time:      "13.00-15.00",
			Room:      "CP9421",
			Subject:   "CP422021",
			CustomID:  "CUSTOM_CP422021",
			Category:  "OUT_OF_SCHEDULE",
		},
	}

	if err := db.AddRound(ctx, roundID, "Purge Round", seats); err != nil {
		t.Fatalf("AddRound failed: %v", err)
	}

	// Purge standard round
	if err := db.PurgeRound(ctx, roundID); err != nil {
		t.Fatalf("PurgeRound failed: %v", err)
	}

	// Only OUT_OF_SCHEDULE custom seat should remain
	remaining, err := db.GetAllSeats(ctx)
	if err != nil {
		t.Fatalf("GetAllSeats failed: %v", err)
	}

	if len(remaining) != 1 {
		t.Fatalf("Expected 1 custom seat remaining after PurgeRound, got %d", len(remaining))
	}

	if remaining[0].Subject != "CP422021" {
		t.Errorf("Expected remaining seat subject CP422021, got %s", remaining[0].Subject)
	}
}

func TestGetSeatsFilterCombination(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_filter.db")

	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	roundID := "filter_round"

	seats := []models.Seat{
		{
			ExamRound: roundID,
			StudentID: "11111",
			Date:      "2026-10-01",
			Time:      "08.30-11.30",
			Room:      "SC201",
			Seat:      "A01",
			Subject:   "MATH101",
		},
		{
			ExamRound: roundID,
			StudentID: "22222",
			Date:      "2026-10-01",
			Time:      "08.30-11.30",
			Room:      "SC201",
			Seat:      "A02",
			Subject:   "MATH101",
		},
	}

	if err := db.AddRound(ctx, roundID, "Filter Round", seats); err != nil {
		t.Fatalf("AddRound failed: %v", err)
	}

	seatOpt := "A02"
	filtered, err := db.GetSeats(ctx, models.ExploreQuery{
		Round: roundID,
		Room:  "SC201",
		Date:  "2026-10-01",
		Time:  "08.30-11.30",
		Seat:  &seatOpt,
	})
	if err != nil {
		t.Fatalf("GetSeats failed: %v", err)
	}

	if len(filtered) != 1 || filtered[0].StudentID != "22222" {
		t.Errorf("Expected single seat record for student 22222, got %v", filtered)
	}
}

func TestForeignKeyConstraints(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_fk.db")

	db, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	validRound := "mid_1_2569"

	// Insert valid round first
	seats := []models.Seat{
		{
			ExamRound: validRound,
			StudentID: "123456789-0",
			Date:      "2026-08-29",
			Time:      "13.00-16.00",
			Room:      "CP9226",
			Subject:   "CP421025",
		},
	}
	if err := db.AddRound(ctx, validRound, "กลางภาค 1/2569", seats); err != nil {
		t.Fatalf("AddRound failed for valid round: %v", err)
	}

	// Purge round info to test ON DELETE CASCADE constraint
	if err := db.PurgeRound(ctx, validRound); err != nil {
		t.Fatalf("PurgeRound failed: %v", err)
	}

	allSeats, err := db.GetAllSeats(ctx)
	if err != nil {
		t.Fatalf("GetAllSeats failed: %v", err)
	}
	if len(allSeats) != 0 {
		t.Fatalf("Expected 0 seats after purging round info, got %d", len(allSeats))
	}
}

func TestGetOptionsWithoutRoomFilter(t *testing.T) {
	db, err := sqlite.New(filepath.Join(t.TempDir(), "test_nofilter.db"))
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	seats := []models.Seat{
		{ExamRound: "r", StudentID: "1", Date: "2026-08-23", Time: "13.00-16.00", Room: "ANY.ROOM", Subject: "CP1"},
	}
	if err := db.AddRound(ctx, "r", "R", seats); err != nil {
		t.Fatalf("AddRound failed: %v", err)
	}

	rooms, err := db.GetOptions(ctx, models.OptionsQuery{Mode: models.OptionRooms, Round: "r", Date: "2026-08-23", Time: "13.00-16.00"})
	if err != nil {
		t.Fatalf("GetOptions failed: %v", err)
	}
	if len(rooms) != 1 || rooms[0] != "ANY.ROOM" {
		t.Errorf("Expected unfiltered rooms [ANY.ROOM], got %v", rooms)
	}

	if _, err := db.GetOptions(ctx, models.OptionsQuery{Mode: "bogus", Round: "r"}); err != models.ErrInvalidOptionMode {
		t.Errorf("Expected ErrInvalidOptionMode, got %v", err)
	}
}

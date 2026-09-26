package services

import (
	"context"
	"errors"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/models"
)

type fakeRepo struct {
	models.ExamRepository
	seats       []models.Seat
	rounds      []models.Round
	err         error
	lastSeatQ   models.SeatQuery
	lastOptions models.OptionsQuery
}

func (f *fakeRepo) GetSeatsByID(ctx context.Context, q models.SeatQuery) ([]models.Seat, error) {
	f.lastSeatQ = q
	return f.seats, f.err
}

func (f *fakeRepo) GetOptions(ctx context.Context, q models.OptionsQuery) ([]string, error) {
	f.lastOptions = q
	return nil, f.err
}

func (f *fakeRepo) GetRounds(ctx context.Context) ([]models.Round, error) {
	return f.rounds, f.err
}

func (f *fakeRepo) GetAllSeats(ctx context.Context) ([]models.Seat, error) {
	return f.seats, f.err
}

func TestExamServiceNormalizesStudentID(t *testing.T) {
	repo := &fakeRepo{}
	round := "mid_1_2569"
	if _, err := NewExamService(repo, nil).StudentSchedule(context.Background(), "653380123-4", &round); err != nil {
		t.Fatal(err)
	}
	if repo.lastSeatQ.StudentID != "6533801234" || *repo.lastSeatQ.Round != round {
		t.Errorf("query = %+v", repo.lastSeatQ)
	}
}

func TestExamServiceOptionsRoomFilter(t *testing.T) {
	repo := &fakeRepo{}
	ctx := context.Background()

	// No rooms configured: every room is offered.
	NewExamService(repo, NewRoomService(models.RoomCatalog{})).Options(ctx, models.OptionsQuery{Mode: models.OptionRooms})
	if repo.lastOptions.RoomFilter != nil {
		t.Error("RoomFilter should be unset when no room layouts are loaded")
	}

	rooms := NewRoomService(models.RoomCatalog{Rooms: map[string]models.Room{"CP.9127": {}}})
	NewExamService(repo, rooms).Options(ctx, models.OptionsQuery{Mode: models.OptionRooms})
	if f := repo.lastOptions.RoomFilter; f == nil || !f("CP.9127") || f("SC.1101") {
		t.Error("RoomFilter should accept only rooms with a loaded layout")
	}
}

func TestRoundServiceListNeverNil(t *testing.T) {
	svc := NewRoundService(&fakeRepo{err: errors.New("db down")})
	if err := svc.Load(context.Background()); err == nil {
		t.Fatal("expected load error")
	}
	if svc.List() == nil {
		t.Error("List must return an empty slice (JSON []), not nil")
	}

	svc = NewRoundService(&fakeRepo{rounds: []models.Round{{ID: "mid_1_2569", Label: "กลางภาค 1/2569"}}})
	if err := svc.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if svc.Labels()["mid_1_2569"] != "กลางภาค 1/2569" {
		t.Errorf("labels = %v", svc.Labels())
	}
}

func TestStatsServiceRefresh(t *testing.T) {
	svc := NewStatsService(&fakeRepo{err: errors.New("db down")})
	if err := svc.Refresh(context.Background(), nil); err == nil || svc.Dashboard() != nil {
		t.Fatal("failed refresh must leave the dashboard unset")
	}

	svc = NewStatsService(&fakeRepo{seats: []models.Seat{{ExamRound: "r", StudentID: "1"}}})
	if err := svc.Refresh(context.Background(), map[string]string{"r": "Round"}); err != nil {
		t.Fatal(err)
	}
	if got := svc.Dashboard().Stats["r"].StudentCount; got != 1 {
		t.Errorf("student count = %d; want 1", got)
	}
}

func TestIngestUnsupportedFile(t *testing.T) {
	_, err := NewIngestService(nil).Ingest(context.Background(), IngestOptions{
		FilePath: "test.txt",
		RoundID:  "test_round",
	})
	if err == nil {
		t.Errorf("Expected error for unsupported extension .txt, got nil")
	}
}

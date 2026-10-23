package stats

import (
	"context"
	"errors"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

type fakeSeats struct {
	seats []domain.Seat
	err   error
}

func (f *fakeSeats) GetAllSeats(ctx context.Context) ([]domain.Seat, error) {
	return f.seats, f.err
}

func TestServiceRefresh(t *testing.T) {
	svc := NewService(&fakeSeats{err: errors.New("db down")})
	if err := svc.Refresh(context.Background(), nil); err == nil || svc.Dashboard() != nil {
		t.Fatal("failed refresh must leave the dashboard unset")
	}

	svc = NewService(&fakeSeats{seats: []domain.Seat{{ExamRound: "r", StudentID: "1"}}})
	if err := svc.Refresh(context.Background(), map[string]string{"r": "Round"}); err != nil {
		t.Fatal(err)
	}
	if got := svc.Dashboard().Stats["r"].StudentCount; got != 1 {
		t.Errorf("student count = %d; want 1", got)
	}
}

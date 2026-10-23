package round

import (
	"context"
	"errors"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

type fakeRepo struct {
	rounds []domain.Round
	err    error
}

func (f *fakeRepo) GetRounds(ctx context.Context) ([]domain.Round, error) {
	return f.rounds, f.err
}

func TestServiceListNeverNil(t *testing.T) {
	svc := NewService(&fakeRepo{err: errors.New("db down")})
	if err := svc.Load(context.Background()); err == nil {
		t.Fatal("expected load error")
	}
	if svc.List() == nil {
		t.Error("List must return an empty slice (JSON []), not nil")
	}

	svc = NewService(&fakeRepo{rounds: []domain.Round{{ID: "mid_1_2569", Label: "กลางภาค 1/2569"}}})
	if err := svc.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if svc.Labels()["mid_1_2569"] != "กลางภาค 1/2569" {
		t.Errorf("labels = %v", svc.Labels())
	}
}

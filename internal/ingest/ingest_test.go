package ingest

import (
	"context"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

type fakeExtractor []domain.Seat

func (f fakeExtractor) Supports(ext string) bool { return true }

func (f fakeExtractor) Extract(string, Params) ([]domain.Seat, error) { return f, nil }

type fakeRepo struct {
	purgedRound    string
	purgedDatasets []string
	added          int
}

func (f *fakeRepo) PurgeRound(ctx context.Context, roundID string) error {
	f.purgedRound = roundID
	return nil
}

func (f *fakeRepo) PurgeCustomDataset(ctx context.Context, roundID, customID string) error {
	f.purgedDatasets = append(f.purgedDatasets, customID)
	return nil
}

func (f *fakeRepo) AddRound(ctx context.Context, roundID, displayName string, seats []domain.Seat) error {
	f.added = len(seats)
	return nil
}

func TestIngestReplacesRound(t *testing.T) {
	repo := &fakeRepo{}
	n, err := NewService(repo, fakeExtractor{{StudentID: "1"}, {StudentID: "2"}}).Ingest(context.Background(), Options{RoundID: "r"})
	if err != nil || n != 2 || repo.purgedRound != "r" || len(repo.purgedDatasets) != 0 || repo.added != 2 {
		t.Errorf("n=%d err=%v repo=%+v", n, err, repo)
	}
}

func TestIngestReplacesOnlyCustomDatasets(t *testing.T) {
	repo := &fakeRepo{}
	seats := fakeExtractor{{StudentID: "1", CustomID: "LAB"}, {StudentID: "2"}}
	if _, err := NewService(repo, seats).Ingest(context.Background(), Options{RoundID: "r"}); err != nil {
		t.Fatal(err)
	}
	if repo.purgedRound != "" || len(repo.purgedDatasets) != 1 || repo.purgedDatasets[0] != "LAB" {
		t.Errorf("repo = %+v", repo)
	}
}

func TestIngestRejectsCommaInLabel(t *testing.T) {
	repo := &fakeRepo{}
	seats := fakeExtractor{{StudentID: "1", Labels: []string{"ok", "a,b"}}}
	if _, err := NewService(repo, seats).Ingest(context.Background(), Options{RoundID: "r"}); err == nil {
		t.Fatal("expected an error for a label containing a comma")
	}
	if repo.purgedRound != "" || len(repo.purgedDatasets) != 0 || repo.added != 0 {
		t.Errorf("rejected import touched the repository: %+v", repo)
	}
}

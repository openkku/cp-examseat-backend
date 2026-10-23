package admin

import (
	"context"
	"errors"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

type fakeRepo struct {
	Repository
	rounds    map[string]string
	summaries []RoundSummary
	deleted   string
}

func (f *fakeRepo) RoundSummaries(ctx context.Context) ([]RoundSummary, error) {
	return f.summaries, nil
}

func (f *fakeRepo) RoundExists(ctx context.Context, id string) (bool, error) {
	_, ok := f.rounds[id]
	return ok, nil
}

func (f *fakeRepo) SetRoundLabel(ctx context.Context, id, label string) error {
	if _, ok := f.rounds[id]; !ok {
		return domain.ErrRoundNotFound
	}
	f.rounds[id] = label
	return nil
}

func (f *fakeRepo) DeleteRound(ctx context.Context, id string) error {
	f.deleted = id
	return nil
}

type fakePurger struct{ purged string }

func (f *fakePurger) PurgeCustomDataset(ctx context.Context, round, customID string) error {
	f.purged = round + "/" + customID
	return nil
}

type fakeImporter struct {
	opts  ingest.Options
	seats []domain.Seat
	err   error
}

func (f *fakeImporter) SupportedExtension(ext string) bool { return ext == ".json" }

func (f *fakeImporter) Ingest(ctx context.Context, opts ingest.Options) (int, error) {
	f.opts = opts
	return 3, nil
}

func (f *fakeImporter) Preview(opts ingest.Options) ([]domain.Seat, error) {
	f.opts = opts
	return f.seats, f.err
}

type fakeReloader struct {
	err   error
	calls int
}

func (f *fakeReloader) Reload(ctx context.Context) error { f.calls++; return f.err }

func newTestService() (*Service, *fakeRepo, *fakePurger, *fakeImporter, *fakeReloader) {
	repo := &fakeRepo{rounds: map[string]string{"mid_1_2569": "old"}}
	purger, importer, reloader := &fakePurger{}, &fakeImporter{}, &fakeReloader{}
	return NewService(repo, purger, importer, reloader), repo, purger, importer, reloader
}

func TestRenameRound(t *testing.T) {
	ctx := context.Background()
	s, repo, _, _, reloader := newTestService()

	var invalidErr *InvalidInputError
	for _, tc := range [][2]string{{"../x", "Label"}, {"mid_1_2569", "  "}, {"mid_1_2569", string(make([]byte, 201))}} {
		if _, err := s.RenameRound(ctx, tc[0], tc[1]); !errors.As(err, &invalidErr) {
			t.Errorf("RenameRound(%q) = %v; want InvalidInputError", tc[0], err)
		}
	}
	if _, err := s.RenameRound(ctx, "nope", "x"); !errors.Is(err, domain.ErrRoundNotFound) {
		t.Errorf("unknown round = %v", err)
	}
	label, err := s.RenameRound(ctx, "mid_1_2569", "  Midterm ")
	if err != nil || label != "Midterm" || repo.rounds["mid_1_2569"] != "Midterm" || reloader.calls != 1 {
		t.Errorf("rename = %q, %v, %v, reloads %d", label, err, repo.rounds, reloader.calls)
	}
}

func TestDeleteRound(t *testing.T) {
	ctx := context.Background()
	s, repo, purger, _, _ := newTestService()

	if err := s.DeleteRound(ctx, "nope", ""); !errors.Is(err, domain.ErrRoundNotFound) {
		t.Errorf("unknown round = %v", err)
	}
	if err := s.DeleteRound(ctx, "mid_1_2569", "LAB X"); err == nil {
		t.Error("unsafe custom ID accepted")
	}
	if err := s.DeleteRound(ctx, "mid_1_2569", "LAB"); err != nil || purger.purged != "mid_1_2569/LAB" || repo.deleted != "" {
		t.Errorf("dataset delete = %v, purged %q, deleted %q", err, purger.purged, repo.deleted)
	}
	if err := s.DeleteRound(ctx, "mid_1_2569", ""); err != nil || repo.deleted != "mid_1_2569" {
		t.Errorf("round delete = %v, deleted %q", err, repo.deleted)
	}
}

func TestImport(t *testing.T) {
	ctx := context.Background()
	s, _, _, importer, reloader := newTestService()

	if err := s.ValidateFileType(".exe"); err == nil {
		t.Error(".exe accepted")
	}
	saved, err := s.Import(ctx, ImportRequest{FilePath: "f.json", Round: "final_1_2569"})
	if err != nil || saved != 3 || importer.opts.DisplayName != "ปลายภาค 1/2569" {
		t.Errorf("import = %d, %v, opts %+v", saved, err, importer.opts)
	}

	reloader.err = errors.New("disk gone")
	var reloadErr *ReloadError
	if _, err := s.Import(ctx, ImportRequest{FilePath: "f.json", Round: "r"}); !errors.As(err, &reloadErr) {
		t.Errorf("reload failure = %v; want ReloadError", err)
	}
}

func TestPreviewImport(t *testing.T) {
	ctx := context.Background()
	s, repo, _, importer, _ := newTestService()

	repo.summaries = []RoundSummary{
		{
			ID:              "mid_1_2569",
			InScheduleSeats: 150,
			CustomDatasets: []CustomDataset{
				{ID: "LAB_A", Seats: 40},
				{ID: "LAB_B", Seats: 60},
			},
		},
	}

	// 1. In-schedule preview
	importer.seats = []domain.Seat{
		{StudentID: "S1", Room: "SC1101", Date: "2026-10-01", Time: "09:00-12:00", Subject: "CP101", Section: "1"},
		{StudentID: "S2", Room: "SC1101", Date: "2026-10-01", Time: "09:00-12:00", Subject: "CP101", Section: "1"},
		{StudentID: "S3", Room: "SC1102", Date: "2026-10-02", Time: "13:00-16:00", Subject: "CP102", Section: "2"},
	}

	prev, err := s.PreviewImport(ctx, ImportRequest{FilePath: "f.json", Round: "mid_1_2569"})
	if err != nil {
		t.Fatalf("PreviewImport error: %v", err)
	}
	if prev.Seats != 3 || prev.Students != 3 || len(prev.Rooms) != 2 || len(prev.Dates) != 2 || prev.Sessions != 2 {
		t.Errorf("unexpected preview counts: %+v", prev)
	}
	if len(prev.Replaces) != 1 || prev.Replaces[0].Type != "in_schedule" || prev.Replaces[0].Seats != 150 {
		t.Errorf("unexpected replaces for in-schedule: %+v", prev.Replaces)
	}
	if len(prev.Rows) != 3 {
		t.Errorf("unexpected rows count: %d", len(prev.Rows))
	}

	// 2. Custom dataset preview (replaces custom dataset)
	importer.seats = []domain.Seat{
		{StudentID: "S1", Room: "SC1101", Date: "2026-10-01", CustomID: "LAB_A"},
		{StudentID: "S4", Room: "SC1101", Date: "2026-10-01", CustomID: "LAB_NEW"},
	}
	prevCustom, err := s.PreviewImport(ctx, ImportRequest{FilePath: "f.json", Round: "mid_1_2569"})
	if err != nil {
		t.Fatalf("PreviewImport custom error: %v", err)
	}
	if len(prevCustom.CustomIDs) != 2 || prevCustom.CustomIDs[0] != "LAB_A" || prevCustom.CustomIDs[1] != "LAB_NEW" {
		t.Errorf("unexpected customIDs: %v", prevCustom.CustomIDs)
	}
	if len(prevCustom.Replaces) != 2 {
		t.Fatalf("unexpected replaces length: %d", len(prevCustom.Replaces))
	}
	if prevCustom.Replaces[0].ID != "LAB_A" || prevCustom.Replaces[0].Seats != 40 {
		t.Errorf("expected LAB_A to replace 40 seats, got %+v", prevCustom.Replaces[0])
	}
	if prevCustom.Replaces[1].ID != "LAB_NEW" || prevCustom.Replaces[1].Seats != 0 {
		t.Errorf("expected LAB_NEW to replace 0 seats, got %+v", prevCustom.Replaces[1])
	}

	// 3. Validation error on bad round
	if _, err := s.PreviewImport(ctx, ImportRequest{FilePath: "f.json", Round: "bad/round"}); err == nil {
		t.Error("expected error for bad round ID")
	}
}

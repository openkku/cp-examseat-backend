package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

type fakeRepo struct {
	Repository
	rounds  map[string]string
	deleted string
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

type fakeImporter struct{ opts ingest.Options }

func (f *fakeImporter) SupportedExtension(ext string) bool { return ext == ".json" }

func (f *fakeImporter) Ingest(ctx context.Context, opts ingest.Options) (int, error) {
	f.opts = opts
	return 3, nil
}

type fakeReloader struct {
	err   error
	calls int
}

func (f *fakeReloader) Reload(ctx context.Context) error { f.calls++; return f.err }
func (f *fakeReloader) LastReload() time.Time            { return time.Time{} }

func newTestService() (*Service, *fakeRepo, *fakePurger, *fakeImporter, *fakeReloader) {
	repo := &fakeRepo{rounds: map[string]string{"mid_1_2569": "old"}}
	purger, importer, reloader := &fakePurger{}, &fakeImporter{}, &fakeReloader{}
	return NewService(repo, purger, importer, nil, reloader), repo, purger, importer, reloader
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

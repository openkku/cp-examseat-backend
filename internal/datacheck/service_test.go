package datacheck_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/datacheck"
	"github.com/openkku/cp-examseat-backend/internal/room"
)

type fakeUsage struct {
	usage []datacheck.RoomSeats
	err   error
}

func (f *fakeUsage) RoomSeats(ctx context.Context) ([]datacheck.RoomSeats, error) {
	return f.usage, f.err
}

type fakeRooms struct {
	meta    map[string]room.Meta
	layouts map[string][]byte
}

func (f *fakeRooms) Rooms() (map[string]room.Meta, error) {
	return f.meta, nil
}

// RawLayout behaves like the file store: a missing file is not an error.
func (f *fakeRooms) RawLayout(file string) ([]byte, bool, error) {
	if file == "unreadable.json" {
		return nil, false, errors.New("permission denied")
	}
	data, ok := f.layouts[file]
	return data, ok, nil
}

type fakeImages struct {
	existing map[string]bool
}

func (f *fakeImages) Exists(path string) bool {
	return f.existing[path]
}

func TestHealthCheckAllProblemKinds(t *testing.T) {
	ctx := context.Background()

	// SC1101: valid layout with seats A1, A2, A3
	// SC1102: invalid layout JSON
	// SC1103: missing layout file
	// SC1104: missing image files
	// SC9999: in metadata but no seats used (unused_room)
	// SC_UNKNOWN: used in round but not in metadata (missing_metadata)

	validLayout := []byte(`{"layout": [{"type": "column", "items": [{"type": "seats", "char": "A", "count": 3}]}]}`)

	rooms := &fakeRooms{
		meta: map[string]room.Meta{
			"SC1101": {LayoutFile: "SC1101.json", LayoutImage: "/room/image/layout/1.webp"},
			"SC1102": {LayoutFile: "SC1102.json"},
			"SC1103": {LayoutFile: "missing.json"},
			"SC1104": {
				LayoutFile:  "SC1101.json",
				LayoutImage: "/room/image/missing_layout.webp",
				Images:      []string{"/room/image/missing_pic.webp", "/room/image/ok.webp", "https://cdn.example/x.webp"},
			},
			"SC9999": {LayoutFile: "SC1101.json"},
		},
		layouts: map[string][]byte{
			"SC1101.json": validLayout,
			"SC1102.json": []byte(`{"layout": [`),
		},
	}

	images := &fakeImages{
		existing: map[string]bool{
			"/room/image/layout/1.webp": true,
			"/room/image/ok.webp":       true,
		},
	}

	usage := &fakeUsage{
		usage: []datacheck.RoomSeats{
			// SC_UNKNOWN: missing_metadata
			{Round: "R1", Room: "SC_UNKNOWN", Seats: []string{"A1"}},
			// SC1101: uses A1, A2, A4, A5, A6 (A4, A5, A6 missing from layout)
			{Round: "R1", Room: "SC1101", Seats: []string{"A1", "A2", "A4", "A5", "A6"}},
			// SC1102: invalid_layout
			{Round: "R1", Room: "SC1102", Seats: []string{"A1"}},
			// SC1103: missing layout
			{Round: "R1", Room: "SC1103", Seats: []string{"A1"}},
			// SC1104: uses A1
			{Round: "R1", Room: "SC1104", Seats: []string{"A1"}},
		},
	}

	svc := datacheck.NewService(usage, rooms, images)
	report, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("svc.Run() error: %v", err)
	}

	kinds := make(map[datacheck.ProblemKind]int)
	for _, p := range report.Problems {
		kinds[p.Kind]++
	}

	// Verify KindMissingMetadata
	if kinds[datacheck.KindMissingMetadata] != 1 {
		t.Errorf("expected 1 missing_metadata problem, got %d", kinds[datacheck.KindMissingMetadata])
	}

	// Verify KindInvalidLayout (SC1102 unparsable, SC1103 missing file)
	if kinds[datacheck.KindInvalidLayout] != 2 {
		t.Errorf("expected 2 invalid_layout problems, got %d", kinds[datacheck.KindInvalidLayout])
	}

	for _, p := range report.Problems {
		if p.Room == "SC1103" {
			if p.Kind != datacheck.KindInvalidLayout || !strings.Contains(p.Message, "missing") {
				t.Errorf("SC1103 (missing layout file): %+v", p)
			}
		}
		if p.Kind == datacheck.KindMissingImage && strings.Contains(p.Message, "https://") {
			t.Errorf("external image URL checked as a local file: %+v", p)
		}
	}

	// Verify KindMissingSeats (SC1101 has 3 missing seats: A4, A5, A6)
	if kinds[datacheck.KindMissingSeats] != 1 {
		t.Errorf("expected 1 missing_seats problem, got %d", kinds[datacheck.KindMissingSeats])
	} else {
		for _, p := range report.Problems {
			if p.Kind == datacheck.KindMissingSeats {
				if len(p.Details) != 3 {
					t.Errorf("expected 3 missing seat details, got %d", len(p.Details))
				}
			}
		}
	}

	// Verify KindMissingImage (SC1104 has 2 missing images: layout_image and 1 in Images)
	if kinds[datacheck.KindMissingImage] != 2 {
		t.Errorf("expected 2 missing_image problems, got %d", kinds[datacheck.KindMissingImage])
	}

	// Verify KindUnusedRoom (SC9999 has no seats in any round)
	if kinds[datacheck.KindUnusedRoom] != 1 {
		t.Errorf("expected 1 unused_room problem, got %d", kinds[datacheck.KindUnusedRoom])
	}
}

func TestHealthCheckSkipsImagesWhenNil(t *testing.T) {
	ctx := context.Background()
	rooms := &fakeRooms{
		meta: map[string]room.Meta{
			"SC1101": {LayoutFile: "SC1101.json", LayoutImage: "/room/image/missing.webp"},
		},
		layouts: map[string][]byte{
			"SC1101.json": []byte(`{"layout": []}`),
		},
	}
	usage := &fakeUsage{
		usage: []datacheck.RoomSeats{
			{Round: "R1", Room: "SC1101"},
		},
	}

	// nil images adapter (e.g. IMAGE_BASE_URL set)
	svc := datacheck.NewService(usage, rooms, nil)
	report, err := svc.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range report.Problems {
		if p.Kind == datacheck.KindMissingImage {
			t.Errorf("expected no image problems when images adapter is nil, got: %+v", p)
		}
	}
}

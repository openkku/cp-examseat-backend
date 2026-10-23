package roomconfig_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
)

func TestParseLayoutAndSeatIDsFixtures(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata", "layouts")
	files, err := filepath.Glob(filepath.Join(testdataDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("no fixture files found in %s", testdataDir)
	}

	for _, jsonFile := range files {
		t.Run(filepath.Base(jsonFile), func(t *testing.T) {
			data, err := os.ReadFile(jsonFile)
			if err != nil {
				t.Fatalf("failed reading %s: %v", jsonFile, err)
			}

			layout, err := roomconfig.ParseLayout(data)
			if err != nil {
				t.Fatalf("ParseLayout(%s) error: %v", jsonFile, err)
			}

			gotSeats := layout.SeatIDs()

			seatsFile := strings.TrimSuffix(jsonFile, ".json") + ".seats.txt"
			expectedData, err := os.ReadFile(seatsFile)
			if err != nil {
				t.Fatalf("failed reading expected seats file %s: %v", seatsFile, err)
			}

			expectedLines := strings.Split(strings.TrimSpace(string(expectedData)), "\n")
			var wantSeats []string
			for _, line := range expectedLines {
				line = strings.TrimSpace(line)
				if line != "" {
					wantSeats = append(wantSeats, line)
				}
			}

			if len(gotSeats) != len(wantSeats) {
				t.Fatalf("seat count mismatch for %s: got %d seats, want %d\ngot:  %v\nwant: %v",
					filepath.Base(jsonFile), len(gotSeats), len(wantSeats), gotSeats, wantSeats)
			}

			for i := range gotSeats {
				if gotSeats[i] != wantSeats[i] {
					t.Errorf("seat[%d] mismatch for %s: got %s, want %s",
						i, filepath.Base(jsonFile), gotSeats[i], wantSeats[i])
				}
			}
		})
	}
}

func TestParseLayoutRejectsInvalidJSON(t *testing.T) {
	_, err := roomconfig.ParseLayout([]byte(`{"layout": [`))
	if err == nil {
		t.Fatal("expected error parsing invalid JSON, got nil")
	}
}

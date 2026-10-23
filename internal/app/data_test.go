package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/database/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/platform/reloadsignal"
	"github.com/openkku/cp-examseat-backend/internal/shared/domain"
)

// Data is managed by the operator app and the migrate CLI, never over the public API.
func TestNoAdminAPI(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "legacy-token")
	srv := newTestServer(t, config.Load())
	for _, path := range []string{"/api/admin/status", "/api/admin/rounds", "/api/admin/import", "/api/admin/reload"} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			req, _ := http.NewRequest(method, srv.URL+path, nil)
			req.Header.Set("Authorization", "Bearer legacy-token")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != http.StatusNotFound && res.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("%s %s = %d; want 404/405", method, path, res.StatusCode)
			}
		}
	}
}

// touchReload signals a reload the way the operator app and migrate do, with
// an mtime that differs even on filesystems with coarse timestamps.
func touchReload(t *testing.T, dataDir string) {
	t.Helper()
	if err := reloadsignal.Touch(dataDir); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(reloadsignal.Path(dataDir), later, later); err != nil {
		t.Fatal(err)
	}
}

// eventually polls path until check passes or 3s have passed.
func eventually(t *testing.T, srv *httptest.Server, path string, check func(response) bool) response {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		res := get(t, srv, path, nil)
		if check(res) || time.Now().After(deadline) {
			return res
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestReloadMarkerPicksUpRoomChanges(t *testing.T) {
	srv := newTestServer(t, config.Config{})
	if res := get(t, srv, "/api/room?room=NEW.1", nil); res.status != http.StatusNotFound {
		t.Fatalf("precondition: %d", res.status)
	}
	dataDir := testDataDir(t, srv)
	writeFile(t, filepath.Join(dataDir, "room", "metadata.json"), `{"NEW.1": {"layout_file": "new.json", "layout_image": "", "map_url": "", "images": []}}`)
	writeFile(t, filepath.Join(dataDir, "room", "map", "new.json"), `{"layout": []}`)
	touchReload(t, dataDir)

	if res := eventually(t, srv, "/api/room?room=NEW.1", func(r response) bool { return r.status == http.StatusOK }); res.status != http.StatusOK {
		t.Errorf("room added on disk not served after .reload: %d %s", res.status, res.body)
	}
}

func TestReloadMarkerPicksUpImports(t *testing.T) {
	srv := newTestServer(t, config.Config{})
	dataDir := testDataDir(t, srv)

	// Warm the caches with "not found" answers for the data about to be imported.
	explore := "/api/explore?round=final_1_2569&room=CP.9127&date=2026-10-01&time=08.30-11.30"
	if res := get(t, srv, explore, nil); res.status != http.StatusNotFound {
		t.Fatalf("precondition: %d", res.status)
	}
	if res := get(t, srv, "/api/calendar/6733809999.ics", nil); res.status != http.StatusNotFound {
		t.Fatalf("precondition: %d", res.status)
	}

	// Another process (the operator app or migrate) writes the database.
	db, err := sqlite.Open(filepath.Join(dataDir, "exams.db"))
	if err != nil {
		t.Fatal(err)
	}
	seats := []domain.Seat{{Sheet: "S9", Date: "2026-10-01", Time: "08.30-11.30", Room: "CP.9127", Subject: "CP9999", SubjectName: "Imported", Section: "1", StudentID: "6733809999", Seat: "A3"}}
	if err := db.AddRound(context.Background(), "final_1_2569", "ปลายภาค 1/2569", seats); err != nil {
		t.Fatal(err)
	}
	db.Close()
	touchReload(t, dataDir)

	if res := eventually(t, srv, "/api/rounds", func(r response) bool { return strings.Contains(r.body, "ปลายภาค 1/2569") }); !strings.Contains(res.body, "ปลายภาค 1/2569") {
		t.Errorf("new round not visible after .reload: %s", res.body)
	}
	if res := eventually(t, srv, explore, func(r response) bool { return r.status == http.StatusOK }); res.status != http.StatusOK {
		t.Errorf("explore cache not cleared: %d %s", res.status, res.body)
	}
	if res := get(t, srv, "/api/calendar/6733809999.ics", nil); res.status != http.StatusOK {
		t.Errorf("calendar cache not cleared: %d", res.status)
	}
	if res := get(t, srv, "/api/stats", nil); !strings.Contains(res.body, "final_1_2569") {
		t.Errorf("stats not refreshed")
	}
}

func TestRateLimiting(t *testing.T) {
	srv := newTestServer(t, config.Config{RateLimitRPS: 0.001, RateLimitBurst: 2})

	for i := 0; i < 2; i++ {
		if res := get(t, srv, "/api/rounds", nil); res.status != http.StatusOK {
			t.Fatalf("request %d = %d", i, res.status)
		}
	}
	res := get(t, srv, "/api/rounds", nil)
	if res.status != http.StatusTooManyRequests || res.header.Get("Retry-After") == "" {
		t.Errorf("over limit = %d Retry-After=%q", res.status, res.header.Get("Retry-After"))
	}
	if res := get(t, srv, "/healthz", nil); res.status != http.StatusOK {
		t.Errorf("health checks must not be rate limited: %d", res.status)
	}
}

// testDataDir returns the data directory registered for srv by newTestServer.
func testDataDir(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	dir, ok := dataDirs.Load(srv.URL)
	if !ok {
		t.Fatal("unknown test server")
	}
	return dir.(string)
}

package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/klauspost/compress/gzip"

	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/repositories/sqlite"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// newTestServer seeds a data directory and serves the fully wired app.
func newTestServer(t *testing.T, cfg config.Config) *httptest.Server {
	t.Helper()
	dataDir := t.TempDir()
	cfg.DataDir = dataDir

	writeFile(t, filepath.Join(dataDir, "room", "metadata.json"), `{
		"CP.9127": {"layout_file": "CP9127.json", "layout_image": "/room/image/CP.9127.jpg", "map_url": "https://maps.test", "images": []},
		"SC.1101": {"layout_file": "SC1101.json", "layout_image": "/room/image/SC.1101.jpg", "map_url": "", "images": []}
	}`)
	writeFile(t, filepath.Join(dataDir, "room", "map", "CP9127.json"), `{"frontLabel": "Board", "layout": [{"type": "column", "items": [{"type": "seats", "char": "A", "count": 3}]}]}`)
	writeFile(t, filepath.Join(dataDir, "room", "map", "SC1101.json"), `{"layout": []}`)
	writeFile(t, filepath.Join(dataDir, "room", "image", "CP.9127.jpg"), "JPEG")
	writeFile(t, filepath.Join(dataDir, "secret.txt"), "SECRET")

	db, err := sqlite.New(cfg.DatabasePath())
	if err != nil {
		t.Fatal(err)
	}
	seats := []models.Seat{
		{Sheet: "S1", Date: "2026-09-01", Time: "08.30-11.30", Room: "CP.9127", Subject: "CP1001", SubjectName: "Intro", Section: "1", StudentID: "6533801234", Seat: "A1", Branch: "CP-CS"},
		{Sheet: "S1", Date: "2026-09-01", Time: "08.30-11.30", Room: "CP.9127", Subject: "CP1001", SubjectName: "Intro", Section: "1", StudentID: "6533801235", Seat: "A2", Branch: "CP-AI"},
		{Sheet: "S2", Date: "2026-09-02", Time: "13.00-16.00", Room: "UNMAPPED", Subject: "SC2002", SubjectName: "Physics", Section: "2", StudentID: "6533801234", Seat: "B1"},
	}
	if err := db.AddRound(context.Background(), "mid_1_2569", "กลางภาค 1/2569", seats); err != nil {
		t.Fatal(err)
	}
	db.Close()

	application, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { application.Close() })

	srv := httptest.NewServer(application.Handler)
	t.Cleanup(srv.Close)
	dataDirs.Store(srv.URL, dataDir)
	return srv
}

// dataDirs maps test server URLs to their data directory.
var dataDirs sync.Map

type response struct {
	status  int
	header  http.Header
	body    string
	decoded any
}

func get(t *testing.T, srv *httptest.Server, path string, headers map[string]string) response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// Disable the transport's transparent gzip handling so encodings can be asserted.
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var reader io.Reader = res.Body
	if res.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		reader = zr
	}
	body, _ := io.ReadAll(reader)

	out := response{status: res.StatusCode, header: res.Header, body: string(body)}
	json.Unmarshal(body, &out.decoded)
	return out
}

func TestAPI(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string // substring
	}{
		{"health", "/healthz", 200, `"status":"ok"`},
		{"rounds", "/api/rounds", 200, `[{"id":"mid_1_2569","label":"กลางภาค 1/2569"}]`},
		{"exam", "/api/exam?id=653380123-4&round=mid_1_2569", 200, `"student_id":"6533801234"`},
		{"exam missing round", "/api/exam?id=6533801234", 400, `Student ID and Round are required`},
		{"exam not found", "/api/exam?id=1&round=mid_1_2569", 404, `No exam schedules found`},
		{"explore", "/api/explore?round=mid_1_2569&room=CP.9127&date=2026-09-01&time=08.30-11.30&seat=A2", 200, `"seat":"A2"`},
		{"explore empty", "/api/explore?round=mid_1_2569&room=CP.9127&date=2030-01-01&time=08.30-11.30", 404, `{"error":"no exams found"}`},
		{"explore missing params", "/api/explore?round=mid_1_2569", 400, `Round, Room, Date, and Time parameters are required`},
		{"options dates skip unmapped rooms", "/api/options?type=dates&round=mid_1_2569", 200, `["2026-09-01"]`},
		{"options rooms", "/api/options?type=rooms&round=mid_1_2569&date=2026-09-01&time=08.30-11.30", 200, `["CP.9127"]`},
		{"options invalid mode", "/api/options?type=bogus&round=mid_1_2569", 400, `invalid mode`},
		{"options none", "/api/options?type=times&round=mid_1_2569&date=2026-09-02", 404, `No options found`},
		{"stats", "/api/stats", 200, `"student_count":2`},
		{"rooms with layout", "/api/room?room=CP.9127", 200, `"frontLabel":"Board"`},
		{"rooms without layout", "/api/room?no_layout=true", 200, `"SC.1101":{"i_images":[]`},
		{"rooms unknown", "/api/room?room=NOPE", 404, `Rooms not found`},
		{"rooms too many", "/api/room?room=a,b,c", 405, `Method Not Allowed`},
		{"image", "/room/image/CP.9127.jpg", 200, `JPEG`},
		{"image missing", "/room/image/nope.jpg", 404, `Image not found`},
		{"calendar", "/api/calendar/653380123-4.ics", 200, `BEGIN:VCALENDAR`},
		{"calendar unknown", "/api/calendar/0000000000.ics", 404, `No exam schedules found for this student`},
		{"calendar invalid", "/api/calendar/abc", 400, `Student ID is required`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := get(t, srv, tc.path, nil)
			if res.status != tc.wantStatus {
				t.Errorf("status = %d; want %d (body %s)", res.status, tc.wantStatus, res.body)
			}
			if !strings.Contains(res.body, tc.wantBody) {
				t.Errorf("body %s does not contain %s", res.body, tc.wantBody)
			}
		})
	}
}

func TestAPIRoomLayoutPayload(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	res := get(t, srv, "/api/room", nil)
	rooms := res.decoded.(map[string]any)
	cp := rooms["CP.9127"].(map[string]any)
	for _, key := range []string{"i_layout", "i_map", "i_images", "layout", "frontLabel"} {
		if _, ok := cp[key]; !ok {
			t.Errorf("CP.9127 payload missing %q: %v", key, cp)
		}
	}
	if _, ok := cp["backLabel"]; ok {
		t.Errorf("backLabel should be absent when the layout file has none")
	}

	res = get(t, srv, "/api/room?no_layout=true", nil)
	cp = res.decoded.(map[string]any)["CP.9127"].(map[string]any)
	if _, ok := cp["layout"]; ok {
		t.Errorf("no_layout=true must omit the layout: %v", cp)
	}
}

func TestAPICompressionAndCaching(t *testing.T) {
	srv := newTestServer(t, config.Config{})
	gz := map[string]string{"Accept-Encoding": "gzip"}

	for _, path := range []string{
		"/api/explore?round=mid_1_2569&room=CP.9127&date=2026-09-01&time=08.30-11.30",
		"/api/explore?round=mid_1_2569&room=CP.9127&date=2030-01-01&time=08.30-11.30",
		"/api/room",
	} {
		first := get(t, srv, path, gz)
		second := get(t, srv, path, gz)
		plain := get(t, srv, path, nil)

		if first.header.Get("Content-Encoding") != "gzip" || second.header.Get("Content-Encoding") != "gzip" {
			t.Errorf("%s: expected gzip responses, got %q / %q", path, first.header.Get("Content-Encoding"), second.header.Get("Content-Encoding"))
		}
		if plain.header.Get("Content-Encoding") != "" {
			t.Errorf("%s: identity request got %q", path, plain.header.Get("Content-Encoding"))
		}
		if first.body != second.body || first.body != plain.body || first.status != plain.status {
			t.Errorf("%s: cached/uncached bodies differ:\n%s\n%s\n%s", path, first.body, second.body, plain.body)
		}
	}
}

func TestAPIImagePathTraversal(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	for _, path := range []string{"/room/image/../../secret.txt", "/room/image/..%2f..%2fsecret.txt", "/room/image/"} {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
		req.URL.Opaque = path // send the path verbatim, without client-side cleaning
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode == http.StatusOK || strings.Contains(string(body), "SECRET") {
			t.Errorf("%s escaped the image directory: %d %s", path, res.StatusCode, body)
		}
	}
}

func TestAPICORS(t *testing.T) {
	srv := newTestServer(t, config.Config{CORSAllowedOrigins: []string{"https://exam.test"}})

	res := get(t, srv, "/api/rounds", map[string]string{"Origin": "https://exam.test"})
	if got := res.header.Get("Access-Control-Allow-Origin"); got != "https://exam.test" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}

	res = get(t, srv, "/api/rounds", map[string]string{"Origin": "https://evil.test"})
	if got := res.header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("unlisted origin got Access-Control-Allow-Origin = %q", got)
	}
}

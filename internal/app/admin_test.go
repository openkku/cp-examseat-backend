package app

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/config"
)

const adminToken = "test-admin-token"

type adminClient struct {
	t   *testing.T
	srv *httptest.Server
}

func (c adminClient) do(method, path string, body io.Reader, contentType string) (int, string) {
	c.t.Helper()
	req, _ := http.NewRequest(method, c.srv.URL+path, body)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

func (c adminClient) upload(fields map[string]string, filename, content string) (int, string) {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	if filename != "" {
		fw, _ := mw.CreateFormFile("file", filename)
		fw.Write([]byte(content))
	}
	mw.Close()
	return c.do(http.MethodPost, "/api/admin/import", &buf, mw.FormDataContentType())
}

const importJSON = `[
 {"Sheet": "S9", "Date": "2026-10-01", "Time": "08.30-11.30", "Room": "CP.9127", "Subject": "CP9999", "SubjectName": "Imported", "Section": "1", "StudentID": "6733809999", "Seat": "A3"}
]`

func TestAdminDisabledWithoutToken(t *testing.T) {
	srv := newTestServer(t, config.Config{})
	if res := get(t, srv, "/api/admin/rounds", map[string]string{"Authorization": "Bearer "}); res.status != http.StatusNotFound {
		t.Errorf("admin API without ADMIN_TOKEN = %d; want 404", res.status)
	}
}

func TestAdminRequiresToken(t *testing.T) {
	srv := newTestServer(t, config.Config{AdminToken: adminToken})
	for _, h := range []map[string]string{nil, {"Authorization": "Bearer wrong"}} {
		if res := get(t, srv, "/api/admin/rounds", h); res.status != http.StatusUnauthorized {
			t.Errorf("admin with %v = %d; want 401", h, res.status)
		}
	}
}

func TestAdminImportReloadsImmediately(t *testing.T) {
	srv := newTestServer(t, config.Config{AdminToken: adminToken, MaxUploadBytes: 1 << 20})
	c := adminClient{t, srv}

	// Warm the caches with "not found" answers for the data we are about to import.
	explore := "/api/explore?round=final_1_2569&room=CP.9127&date=2026-10-01&time=08.30-11.30"
	if res := get(t, srv, explore, nil); res.status != http.StatusNotFound {
		t.Fatalf("precondition: %d", res.status)
	}
	if res := get(t, srv, "/api/calendar/6733809999.ics", nil); res.status != http.StatusNotFound {
		t.Fatalf("precondition: %d", res.status)
	}

	status, body := c.upload(map[string]string{"round": "final_1_2569", "display": "ปลายภาค 1/2569"}, "seats.json", importJSON)
	if status != http.StatusOK || !strings.Contains(body, `"seats":1`) {
		t.Fatalf("import = %d %s", status, body)
	}

	if res := get(t, srv, "/api/rounds", nil); !strings.Contains(res.body, "ปลายภาค 1/2569") {
		t.Errorf("new round not visible without restart: %s", res.body)
	}
	if res := get(t, srv, explore, nil); res.status != http.StatusOK {
		t.Errorf("explore cache not cleared after import: %d %s", res.status, res.body)
	}
	if res := get(t, srv, "/api/calendar/6733809999.ics", nil); res.status != http.StatusOK {
		t.Errorf("calendar cache not cleared after import: %d", res.status)
	}
	if res := get(t, srv, "/api/stats", nil); !strings.Contains(res.body, "final_1_2569") {
		t.Errorf("stats not refreshed after import")
	}

	status, body = c.do(http.MethodGet, "/api/admin/rounds", nil, "")
	var rounds []map[string]any
	json.Unmarshal([]byte(body), &rounds)
	if status != http.StatusOK || len(rounds) != 2 {
		t.Fatalf("admin rounds = %d %s", status, body)
	}
}

func TestAdminImportValidation(t *testing.T) {
	srv := newTestServer(t, config.Config{AdminToken: adminToken, MaxUploadBytes: 1 << 10})
	c := adminClient{t, srv}

	for _, tc := range []struct {
		name     string
		fields   map[string]string
		filename string
		content  string
		want     int
	}{
		{"missing round", map[string]string{}, "a.json", importJSON, http.StatusBadRequest},
		{"unsafe round id", map[string]string{"round": "../../x"}, "a.json", importJSON, http.StatusBadRequest},
		{"unsafe custom id", map[string]string{"round": "r", "custom_id": "a b"}, "a.json", importJSON, http.StatusBadRequest},
		{"missing file", map[string]string{"round": "r"}, "", "", http.StatusBadRequest},
		{"unsupported type", map[string]string{"round": "r"}, "a.exe", "MZ", http.StatusBadRequest},
		{"unparsable file", map[string]string{"round": "r"}, "a.json", "{not json", http.StatusUnprocessableEntity},
		{"too large", map[string]string{"round": "r"}, "a.json", strings.Repeat(" ", 4<<10), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := c.upload(tc.fields, tc.filename, tc.content)
			if status != tc.want {
				t.Errorf("= %d %s; want %d", status, body, tc.want)
			}
			if strings.Contains(body, os.TempDir()) {
				t.Errorf("error leaks the server temp path: %s", body)
			}
		})
	}
}

func TestAdminRenameDeleteAndBackups(t *testing.T) {
	backupDir := filepath.Join(t.TempDir(), "backups")
	srv := newTestServer(t, config.Config{AdminToken: adminToken, MaxUploadBytes: 1 << 20, BackupDir: backupDir, BackupKeep: 10})
	c := adminClient{t, srv}

	c.upload(map[string]string{"round": "mid_1_2569", "custom_id": "LAB_X"}, "lab.json", importJSON)

	status, _ := c.do(http.MethodPatch, "/api/admin/rounds/mid_1_2569", strings.NewReader(`{"label":"Midterm"}`), "application/json")
	if status != http.StatusOK {
		t.Fatalf("rename = %d", status)
	}
	if res := get(t, srv, "/api/rounds", nil); !strings.Contains(res.body, "Midterm") {
		t.Errorf("rename not visible: %s", res.body)
	}
	if status, _ := c.do(http.MethodPatch, "/api/admin/rounds/nope", strings.NewReader(`{"label":"x"}`), "application/json"); status != http.StatusNotFound {
		t.Errorf("rename unknown = %d", status)
	}

	// Deleting a custom dataset keeps the rest of the round.
	if status, body := c.do(http.MethodDelete, "/api/admin/rounds/mid_1_2569?custom_id=LAB_X", nil, ""); status != http.StatusOK {
		t.Fatalf("delete dataset = %d %s", status, body)
	}
	if res := get(t, srv, "/api/exam?id=6733809999&round=mid_1_2569", nil); res.status != http.StatusNotFound {
		t.Errorf("custom dataset still served: %d", res.status)
	}
	if res := get(t, srv, "/api/exam?id=6533801234&round=mid_1_2569", nil); res.status != http.StatusOK {
		t.Errorf("in-schedule seats lost with the dataset: %d", res.status)
	}

	if status, _ := c.do(http.MethodDelete, "/api/admin/rounds/mid_1_2569", nil, ""); status != http.StatusOK {
		t.Fatalf("delete round = %d", status)
	}
	if res := get(t, srv, "/api/rounds", nil); strings.TrimSpace(res.body) != "[]" {
		t.Errorf("rounds after delete = %s", res.body)
	}
	if status, _ := c.do(http.MethodDelete, "/api/admin/rounds/mid_1_2569", nil, ""); status != http.StatusNotFound {
		t.Errorf("delete missing round = %d", status)
	}

	// Import and both deletes each took a safety backup first.
	entries, _ := os.ReadDir(backupDir)
	if len(entries) != 3 {
		t.Errorf("safety backups = %d; want 3", len(entries))
	}

	if status, body := c.do(http.MethodPost, "/api/admin/backups", nil, ""); status != http.StatusCreated {
		t.Errorf("create backup = %d %s", status, body)
	}
	status, body := c.do(http.MethodGet, "/api/admin/backup", nil, "")
	if status != http.StatusOK || !strings.HasPrefix(body, "SQLite format 3") {
		t.Errorf("backup download = %d %q", status, body[:min(len(body), 20)])
	}

	status, body = c.do(http.MethodGet, "/api/admin/status", nil, "")
	if status != http.StatusOK || !strings.Contains(body, `"backups_enabled":true`) {
		t.Errorf("status = %d %s", status, body)
	}
}

func TestReloadPicksUpRoomChanges(t *testing.T) {
	cfg := config.Config{AdminToken: adminToken}
	srv := newTestServer(t, cfg)
	c := adminClient{t, srv}

	// newTestServer writes the data dir; find it via the admin status of rooms.
	if res := get(t, srv, "/api/room?room=NEW.1", nil); res.status != http.StatusNotFound {
		t.Fatalf("precondition: %d", res.status)
	}
	dataDir := testDataDir(t, srv)
	writeFile(t, filepath.Join(dataDir, "room", "metadata.json"), `{"NEW.1": {"layout_file": "new.json", "layout_image": "", "map_url": "", "images": []}}`)
	writeFile(t, filepath.Join(dataDir, "room", "map", "new.json"), `{"layout": []}`)

	if status, _ := c.do(http.MethodPost, "/api/admin/reload", nil, ""); status != http.StatusOK {
		t.Fatalf("reload = %d", status)
	}
	if res := get(t, srv, "/api/room?room=NEW.1", nil); res.status != http.StatusOK {
		t.Errorf("room added on disk not served after reload: %d %s", res.status, res.body)
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

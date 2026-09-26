package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/controllers"
	"github.com/openkku/cp-examseat-backend/internal/repositories/filesystem"
)

// The room-config tool writes files on disk, so a web page the operator
// happens to visit must not be able to drive it (CSRF / DNS rebinding).

func newRoomConfigHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	dir := t.TempDir()
	store := filesystem.NewRoomConfigStore(dir)
	ui := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ui")) }
	return NewRoomConfig(controllers.NewRoomConfigController(store), ui), dir
}

func roomConfigRequest(method, path, body string, headers map[string]string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Host = "localhost:8081"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRoomConfigAcceptsSameOriginJSONWrites(t *testing.T) {
	h, dir := newRoomConfigHandler(t)

	rec := serve(h, roomConfigRequest(http.MethodPost, "/api/config/CP.9127", `{"layout_file":"CP9127.json"}`, map[string]string{
		"Content-Type": "application/json",
		"Origin":       "http://localhost:8081",
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("legit write = %d %s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(dir, "map", "CP9127.json")); err != nil {
		t.Errorf("layout not created: %v", err)
	}

	// The Vite dev server (npm run dev) proxies with the browser's Origin.
	rec = serve(h, roomConfigRequest(http.MethodPost, "/api/layout/CP9127.json", `{"layout":[]}`, map[string]string{
		"Content-Type": "application/json; charset=utf-8",
		"Origin":       "http://localhost:5174",
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("write from the Vite dev server = %d %s", rec.Code, rec.Body)
	}

	// curl and scripts send no Origin.
	rec = serve(h, roomConfigRequest(http.MethodDelete, "/api/config/CP.9127", "", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("delete without Origin = %d %s", rec.Code, rec.Body)
	}
}

func TestRoomConfigRejectsCrossSiteSimpleRequest(t *testing.T) {
	h, dir := newRoomConfigHandler(t)

	// A <form> or fetch(..., {mode: 'no-cors'}) from another site sends a
	// "simple" request (text/plain, no preflight) with a foreign Origin.
	for _, headers := range []map[string]string{
		{"Content-Type": "text/plain", "Origin": "https://evil.example"},
		{"Content-Type": "application/json", "Origin": "https://evil.example"},
		{"Content-Type": "application/json", "Origin": "http://localhost.evil.example"},
		{"Content-Type": "text/plain"},
	} {
		rec := serve(h, roomConfigRequest(http.MethodPost, "/api/layout/pwn.json", `{"layout":[]}`, headers))
		if rec.Code < 400 {
			t.Errorf("cross-site write %v accepted: %d", headers, rec.Code)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "map", "pwn.json")); !os.IsNotExist(err) {
		t.Errorf("cross-site request wrote a file")
	}

	rec := serve(h, roomConfigRequest(http.MethodDelete, "/api/config/X", "", map[string]string{"Origin": "https://evil.example"}))
	if rec.Code != http.StatusForbidden {
		t.Errorf("cross-site delete = %d; want 403", rec.Code)
	}
}

func TestRoomConfigNoWildcardCORS(t *testing.T) {
	h, _ := newRoomConfigHandler(t)

	rec := serve(h, roomConfigRequest(http.MethodOptions, "/api/config/X", "", map[string]string{
		"Origin":                        "https://evil.example",
		"Access-Control-Request-Method": "POST",
	}))
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("preflight from another site allowed: Access-Control-Allow-Origin = %q", got)
	}

	rec = serve(h, roomConfigRequest(http.MethodGet, "/api/config", "", map[string]string{"Origin": "https://evil.example"}))
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("config readable cross-origin: Access-Control-Allow-Origin = %q", got)
	}
}

func TestRoomConfigRejectsForeignHost(t *testing.T) {
	h, _ := newRoomConfigHandler(t)

	// DNS rebinding: evil.example resolves to 127.0.0.1 but keeps its Host.
	req := roomConfigRequest(http.MethodGet, "/api/config", "", nil)
	req.Host = "evil.example:8081"
	if rec := serve(h, req); rec.Code != http.StatusForbidden {
		t.Errorf("foreign Host = %d; want 403", rec.Code)
	}

	for _, host := range []string{"localhost:8081", "127.0.0.1:8081", "[::1]:8081", "localhost"} {
		req := roomConfigRequest(http.MethodGet, "/api/config", "", nil)
		req.Host = host
		if rec := serve(h, req); rec.Code != http.StatusOK {
			t.Errorf("Host %s = %d; want 200", host, rec.Code)
		}
	}
}

func TestRoomConfigLimitsBodySize(t *testing.T) {
	h, dir := newRoomConfigHandler(t)

	body := `{"layout":["` + strings.Repeat("x", 2<<20) + `"]}`
	rec := serve(h, roomConfigRequest(http.MethodPost, "/api/layout/big.json", body, map[string]string{"Content-Type": "application/json"}))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("2 MiB body = %d; want 413", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(dir, "map", "big.json")); !os.IsNotExist(err) {
		t.Errorf("oversized layout was written")
	}
}

func TestRoomConfigLayoutTraversal(t *testing.T) {
	h, dir := newRoomConfigHandler(t)

	rec := serve(h, roomConfigRequest(http.MethodPost, "/api/layout/..%2f..%2fescape.json", `{"layout":[]}`, map[string]string{"Content-Type": "application/json"}))
	if rec.Code >= 500 {
		t.Fatalf("traversal attempt = %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.json")); !os.IsNotExist(err) {
		t.Errorf("layout written outside map/")
	}
}

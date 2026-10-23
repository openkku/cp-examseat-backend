package operator_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The operator app changes data, so a web page the operator happens to visit
// must not be able to drive it (CSRF / DNS rebinding).

func TestAcceptsSameOriginWrites(t *testing.T) {
	o := newOperator(t, 1<<20)

	rec := o.serve(request(http.MethodPost, "/api/rooms/CP.9127", `{"layout_file":"CP9127.json"}`, map[string]string{
		"Content-Type": "application/json",
		"Origin":       "http://localhost:8081",
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("legit write = %d %s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(o.dataDir, "room", "map", "CP9127.json")); err != nil {
		t.Errorf("layout not created: %v", err)
	}

	// The Vite dev server (npm run dev) proxies with the browser's Origin.
	rec = o.serve(request(http.MethodPost, "/api/rooms/layouts/CP9127.json", `{"layout":[]}`, map[string]string{
		"Content-Type": "application/json; charset=utf-8",
		"Origin":       "http://localhost:5174",
	}))
	if rec.Code != http.StatusOK {
		t.Errorf("write from the Vite dev server = %d %s", rec.Code, rec.Body)
	}

	// curl and scripts send no Origin.
	rec = o.serve(request(http.MethodDelete, "/api/rooms/CP.9127", "", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("delete without Origin = %d %s", rec.Code, rec.Body)
	}
}

func TestRejectsCrossSiteSimpleRequest(t *testing.T) {
	o := newOperator(t, 1<<20)

	// A <form> or fetch(..., {mode: 'no-cors'}) from another site sends a
	// "simple" request (text/plain, no preflight) with a foreign Origin.
	for _, headers := range []map[string]string{
		{"Content-Type": "text/plain", "Origin": "https://evil.example"},
		{"Content-Type": "application/json", "Origin": "https://evil.example"},
		{"Content-Type": "application/json", "Origin": "http://localhost.evil.example"},
		{"Content-Type": "text/plain"},
	} {
		rec := o.serve(request(http.MethodPost, "/api/rooms/layouts/pwn.json", `{"layout":[]}`, headers))
		if rec.Code < 400 {
			t.Errorf("cross-site write %v accepted: %d", headers, rec.Code)
		}
	}
	if _, err := os.Stat(filepath.Join(o.dataDir, "room", "map", "pwn.json")); !os.IsNotExist(err) {
		t.Errorf("cross-site request wrote a file")
	}

	for _, path := range []string{"/api/rooms/X", "/api/rounds/mid_1_2569"} {
		rec := o.serve(request(http.MethodDelete, path, "", map[string]string{"Origin": "https://evil.example"}))
		if rec.Code != http.StatusForbidden {
			t.Errorf("cross-site DELETE %s = %d; want 403", path, rec.Code)
		}
	}
}

func TestImportNeedsOperatorHeader(t *testing.T) {
	o := newOperator(t, 1<<20)

	// A cross-site <form enctype="multipart/form-data"> can't add X-Operator.
	for _, headers := range []map[string]string{
		{},
		{"Origin": "https://evil.example", "X-Operator": "1"},
		{"Origin": "https://evil.example"},
	} {
		rec := o.upload(map[string]string{"round": "final_1_2569"}, "a.json", importJSON, headers)
		if rec.Code < 400 {
			t.Errorf("import with %v accepted: %d", headers, rec.Code)
		}
		recPrev := o.uploadTo("/api/rounds/import/preview", map[string]string{"round": "final_1_2569"}, "a.json", importJSON, headers)
		if recPrev.Code < 400 {
			t.Errorf("preview import with %v accepted: %d", headers, recPrev.Code)
		}
	}
	if rec := o.json(http.MethodGet, "/api/rounds/", ""); strings.Contains(rec.Body.String(), "final_1_2569") {
		t.Errorf("refused import still saved a round")
	}
}

func TestNoWildcardCORS(t *testing.T) {
	o := newOperator(t, 1<<20)

	rec := o.serve(request(http.MethodOptions, "/api/rounds/import", "", map[string]string{
		"Origin":                         "https://evil.example",
		"Access-Control-Request-Method":  "POST",
		"Access-Control-Request-Headers": "x-operator",
	}))
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("preflight from another site allowed: Access-Control-Allow-Origin = %q", got)
	}

	rec = o.serve(request(http.MethodGet, "/api/rooms/", "", map[string]string{"Origin": "https://evil.example"}))
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("rooms readable cross-origin: Access-Control-Allow-Origin = %q", got)
	}
}

func TestRejectsForeignHost(t *testing.T) {
	o := newOperator(t, 1<<20)

	// DNS rebinding: evil.example resolves to 127.0.0.1 but keeps its Host.
	req := request(http.MethodGet, "/api/rounds/", "", nil)
	req.Host = "evil.example:8081"
	if rec := o.serve(req); rec.Code != http.StatusForbidden {
		t.Errorf("foreign Host = %d; want 403", rec.Code)
	}

	for _, host := range []string{"localhost:8081", "127.0.0.1:8081", "[::1]:8081", "localhost"} {
		req := request(http.MethodGet, "/api/rooms/", "", nil)
		req.Host = host
		if rec := o.serve(req); rec.Code != http.StatusOK {
			t.Errorf("Host %s = %d; want 200", host, rec.Code)
		}
	}
}

func TestLimitsBodySize(t *testing.T) {
	o := newOperator(t, 1<<20)

	body := `{"layout":["` + strings.Repeat("x", 2<<20) + `"]}`
	rec := o.json(http.MethodPost, "/api/rooms/layouts/big.json", body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("2 MiB layout = %d; want 413", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(o.dataDir, "room", "map", "big.json")); !os.IsNotExist(err) {
		t.Errorf("oversized layout was written")
	}
}

func TestLayoutTraversal(t *testing.T) {
	o := newOperator(t, 1<<20)

	rec := o.json(http.MethodPost, "/api/rooms/layouts/..%2f..%2fescape.json", `{"layout":[]}`)
	if rec.Code >= 500 {
		t.Fatalf("traversal attempt = %d", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(o.dataDir, "escape.json")); !os.IsNotExist(err) {
		t.Errorf("layout written outside map/")
	}
}

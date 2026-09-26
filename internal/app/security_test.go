package app

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/openkku/cp-examseat-backend/internal/config"
)

// Security regression tests for the public API. Each test documents the
// attack it guards against.

func TestSecuritySQLInjectionIsInert(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	payloads := []string{
		`' OR '1'='1`,
		`' OR 1=1 --`,
		`"; DROP TABLE exam_seats; --`,
		`mid_1_2569' UNION SELECT sql,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1 FROM sqlite_master --`,
	}

	for _, p := range payloads {
		q := url.QueryEscape(p)
		for _, path := range []string{
			"/api/exam?id=6533801234&round=" + q,
			"/api/explore?round=" + q + "&room=CP.9127&date=2026-09-01&time=08.30-11.30",
			"/api/explore?round=mid_1_2569&room=" + q + "&date=2026-09-01&time=08.30-11.30",
			"/api/explore?round=mid_1_2569&room=CP.9127&date=2026-09-01&time=08.30-11.30&seat=" + q,
			"/api/options?type=rooms&round=" + q,
			"/api/options?type=rooms&round=mid_1_2569&date=" + q + "&time=" + q,
		} {
			res := get(t, srv, path, nil)
			if res.status == http.StatusOK || res.status >= 500 {
				t.Errorf("%s: status %d (injection must neither match data nor break the query): %s", path, res.status, res.body)
			}
			if strings.Contains(res.body, "CREATE TABLE") || strings.Contains(res.body, "student_id") {
				t.Errorf("%s leaked data: %s", path, res.body)
			}
		}
	}

	// The data must still be intact afterwards.
	if res := get(t, srv, "/api/exam?id=6533801234&round=mid_1_2569", nil); res.status != http.StatusOK {
		t.Fatalf("seats were damaged by injection attempts: %d %s", res.status, res.body)
	}
}

func TestSecurityOversizedParametersAreRejected(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	// Every distinct parameter value becomes a cache key; unbounded values
	// would let a client fill memory with 10k huge cache entries.
	long := strings.Repeat("9", 4096)
	for _, path := range []string{
		"/api/exam?id=" + long + "&round=mid_1_2569",
		"/api/calendar/" + long + ".ics",
		"/api/explore?round=mid_1_2569&room=" + long + "&date=2026-09-01&time=08.30-11.30",
		"/api/explore?round=mid_1_2569&room=CP.9127&date=2026-09-01&time=08.30-11.30&seat=" + long,
		"/api/options?type=dates&round=" + long,
		"/api/room?room=" + long,
	} {
		res := get(t, srv, path, nil)
		if res.status != http.StatusBadRequest {
			t.Errorf("%s...: status %d; want 400", path[:40], res.status)
		}
	}
}

func TestSecurityCalendarFilenameInjection(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	// A quote or header delimiter in the ID must not break out of the
	// Content-Disposition filename.
	res := get(t, srv, "/api/calendar/"+url.PathEscape(`653380123-4";x=".ics`), nil)
	if res.status != http.StatusOK {
		t.Fatalf("status %d: %s", res.status, res.body)
	}
	if got, want := res.header.Get("Content-Disposition"), `attachment; filename="exams-653380123-4.ics"`; got != want {
		t.Errorf("Content-Disposition = %q; want %q", got, want)
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	for _, path := range []string{"/api/rounds", "/api/exam?id=1&round=x", "/room/image/CP.9127.jpg", "/api/calendar/6533801234"} {
		res := get(t, srv, path, nil)
		for header, want := range map[string]string{
			"X-Content-Type-Options":  "nosniff",
			"X-Frame-Options":         "DENY",
			"Referrer-Policy":         "no-referrer",
			"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		} {
			if got := res.header.Get(header); got != want {
				t.Errorf("%s: %s = %q; want %q", path, header, got, want)
			}
		}
	}
}

func TestSecurityWriteMethodsAreNotRouted(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		req, _ := http.NewRequest(method, srv.URL+"/api/exam?id=6533801234&round=mid_1_2569", strings.NewReader("{}"))
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/exam = %d; want 405", method, res.StatusCode)
		}
	}
}

func TestSecurityInternalErrorsAreNotLeaked(t *testing.T) {
	srv := newTestServer(t, config.Config{})

	res := get(t, srv, "/api/options?type=bogus&round=mid_1_2569", nil)
	for _, leak := range []string{"sqlite", "SELECT", "no such", ".go:"} {
		if strings.Contains(res.body, leak) {
			t.Errorf("error body leaks internals (%q): %s", leak, res.body)
		}
	}
}

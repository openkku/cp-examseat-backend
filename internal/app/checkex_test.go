package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/config"
)

// fakeScrapler answers lookups like scrapler: found for 6833800001, queued for
// 6833800002, not_found for everything else.
func fakeScrapler(t *testing.T, calls *atomic.Int32, clients *[]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req struct {
			StudentID string `json:"student_id"`
			Client    string `json:"client"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		*clients = append(*clients, req.Client)
		switch req.StudentID {
		case "6833800001":
			_, _ = w.Write([]byte(`{"status":"found","seats":[{"sheet":"20 ต.ค. 69 (เช้า)","date":"2026-10-20","time_start":"08.30","time_end":"11.30","room":"SC.7401","subject":"CP422022","subject_name":"Database","section":"02","seat":"B51","note":"","branch":"CP-Cy"}]}`))
		case "6833800002":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"status":"queued","position":3}`))
		default:
			_, _ = w.Write([]byte(`{"status":"not_found"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestAPICheckexLookups(t *testing.T) {
	var calls atomic.Int32
	var clients []string
	scrapler := fakeScrapler(t, &calls, &clients)

	srv := newTestServer(t, config.Config{
		TrustedProxies: []string{"loopback"},
		Checkex: config.Checkex{
			ScraplerURL:    scrapler.URL,
			Rounds:         []string{"final_1_2569"},
			RoundLabels:    map[string]string{},
			IDPrefixes:     []string{"68", "69"},
			NotFoundTTL:    time.Hour,
			NotFoundPerIP:  2,
			NotFoundWindow: time.Hour,
		},
	})
	visitor := map[string]string{"X-Forwarded-For": "203.0.113.7"}

	if res := get(t, srv, "/api/rounds", nil); !strings.Contains(res.body, `{"id":"final_1_2569","label":"ปลายภาค 1/2569"}`) {
		t.Fatalf("checkex round not registered: %s", res.body)
	}

	// Found: fetched once, then served from the database.
	for range 2 {
		res := get(t, srv, "/api/exam?id=683380000-1&round=final_1_2569", visitor)
		if res.status != 200 || !strings.Contains(res.body, `"seat":"B51"`) {
			t.Fatalf("found lookup = %d %s", res.status, res.body)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 scrapler call, got %d", calls.Load())
	}
	if clients[0] != "203.0.113.7" {
		t.Fatalf("scrapler got client %q, want the visitor's IP", clients[0])
	}

	// Queued: 202 with the position and a Retry-After header.
	res := get(t, srv, "/api/exam?id=6833800002&round=final_1_2569", visitor)
	if res.status != 202 || res.header.Get("Retry-After") != "5" || !strings.Contains(res.body, `"position":3`) {
		t.Fatalf("queued lookup = %d %v %s", res.status, res.header, res.body)
	}

	// Not found counts against the visitor; the third unknown ID is refused upstream.
	for _, id := range []string{"6833800010", "6833800011"} {
		if res := get(t, srv, "/api/exam?id="+id+"&round=final_1_2569", visitor); res.status != 404 {
			t.Fatalf("not-found lookup = %d %s", res.status, res.body)
		}
	}
	before := calls.Load()
	if res := get(t, srv, "/api/exam?id=6833800012&round=final_1_2569", visitor); res.status != 429 {
		t.Fatalf("limited lookup = %d %s", res.status, res.body)
	}
	if calls.Load() != before {
		t.Fatal("limited lookup reached scrapler")
	}
	// Known students still work for the limited visitor, and other visitors are unaffected.
	if res := get(t, srv, "/api/exam?id=6833800001&round=final_1_2569", visitor); res.status != 200 {
		t.Fatalf("known student for limited visitor = %d", res.status)
	}
	if res := get(t, srv, "/api/exam?id=6833800012&round=final_1_2569", map[string]string{"X-Forwarded-For": "198.51.100.1"}); res.status != 404 {
		t.Fatalf("other visitor = %d %s", res.status, res.body)
	}

	// Rounds not served by checkex never go upstream.
	before = calls.Load()
	if res := get(t, srv, "/api/exam?id=6833800020&round=mid_1_2569", visitor); res.status != 404 || calls.Load() != before {
		t.Fatalf("non-checkex round = %d, calls %d -> %d", res.status, before, calls.Load())
	}

	// The calendar feed includes fetched seats.
	if res := get(t, srv, "/api/calendar/6833800001.ics", nil); res.status != 200 || !strings.Contains(res.body, "CP422022") {
		t.Fatalf("calendar = %d %s", res.status, res.body)
	}
}

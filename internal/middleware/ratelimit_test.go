package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIP(t *testing.T) {
	r := NewClientIPResolver([]string{"loopback", "private", "203.0.113.7"})

	tests := []struct {
		name   string
		remote string
		xff    []string
		want   string
	}{
		{"direct client", "198.51.100.1:5000", nil, "198.51.100.1"},
		{"direct client cannot spoof", "198.51.100.1:5000", []string{"1.2.3.4"}, "198.51.100.1"},
		{"behind trusted proxy", "10.0.0.5:5000", []string{"198.51.100.9"}, "198.51.100.9"},
		{"spoofed hop left of real client", "10.0.0.5:5000", []string{"1.2.3.4, 198.51.100.9"}, "198.51.100.9"},
		{"chain of trusted proxies", "127.0.0.1:5000", []string{"198.51.100.9, 203.0.113.7", "172.16.0.2"}, "198.51.100.9"},
		{"trusted proxy without header", "[::1]:5000", nil, "::1"},
		{"garbage header", "10.0.0.5:5000", []string{"not-an-ip"}, "10.0.0.5"},
		{"ipv6 client", "[fd00::1]:5000", []string{"2001:db8::1"}, "2001:db8::1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remote
			for _, v := range tc.xff {
				req.Header.Add("X-Forwarded-For", v)
			}
			if got := r.ClientIP(req); got != tc.want {
				t.Errorf("ClientIP = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	rl := NewRateLimiter(1, 3, NewClientIPResolver(nil))
	rl.now = func() time.Time { return now }
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))

	do := func(ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/api/rounds", nil)
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	for i := 0; i < 3; i++ {
		if rec := do("198.51.100.1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d within burst = %d", i, rec.Code)
		}
	}
	rec := do("198.51.100.1")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("over burst = %d Retry-After=%q", rec.Code, rec.Header().Get("Retry-After"))
	}

	// Other clients have their own bucket.
	if rec := do("198.51.100.2"); rec.Code != http.StatusOK {
		t.Errorf("second client throttled: %d", rec.Code)
	}

	// Tokens refill over time; a rejected request does not consume one.
	now = now.Add(time.Second)
	if rec := do("198.51.100.1"); rec.Code != http.StatusOK {
		t.Errorf("after refill = %d", rec.Code)
	}

	// Idle buckets are swept.
	now = now.Add(idleTTL + 2*time.Minute)
	do("198.51.100.3")
	if len(rl.visitors) != 1 {
		t.Errorf("idle visitors not swept: %d left", len(rl.visitors))
	}
}

func TestRateLimitDisabled(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RateLimit(0, 0, NewClientIPResolver(nil))(ok)
	for i := 0; i < 100; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("disabled limiter throttled request %d", i)
		}
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })

	tests := []struct {
		name       string
		origins    []string
		origin     string
		method     string
		wantOrigin string
		wantStatus int
	}{
		{"disabled", nil, "https://a.test", http.MethodGet, "", http.StatusTeapot},
		{"wildcard", []string{"*"}, "https://a.test", http.MethodGet, "*", http.StatusTeapot},
		{"listed origin", []string{"https://a.test"}, "https://a.test", http.MethodGet, "https://a.test", http.StatusTeapot},
		{"unlisted origin", []string{"https://a.test"}, "https://b.test", http.MethodGet, "", http.StatusTeapot},
		{"preflight", []string{"*"}, "https://a.test", http.MethodOptions, "*", http.StatusNoContent},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/rounds", nil)
			req.Header.Set("Origin", tc.origin)
			if tc.method == http.MethodOptions {
				req.Header.Set("Access-Control-Request-Method", "GET")
			}
			rec := httptest.NewRecorder()
			CORS(tc.origins)(ok).ServeHTTP(rec, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tc.wantOrigin {
				t.Errorf("Access-Control-Allow-Origin = %q; want %q", got, tc.wantOrigin)
			}
			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestLoopbackDetection(t *testing.T) {
	for host, want := range map[string]bool{
		"localhost:8081":         true,
		"127.0.0.1":              true,
		"127.8.9.1:80":           true,
		"[::1]:8081":             true,
		"localhost.evil.example": false,
		"evil.example:8081":      false,
		"0.0.0.0:8081":           false,
		"192.168.1.10:8081":      false,
	} {
		if got := isLoopbackHost(host); got != want {
			t.Errorf("isLoopbackHost(%q) = %v; want %v", host, got, want)
		}
	}
	for origin, want := range map[string]bool{
		"http://localhost:5174":         true,
		"https://127.0.0.1":             true,
		"http://localhost.evil.example": false,
		"null":                          false,
		"file://localhost":              false,
	} {
		if got := isLoopbackOrigin(origin); got != want {
			t.Errorf("isLoopbackOrigin(%q) = %v; want %v", origin, got, want)
		}
	}
}

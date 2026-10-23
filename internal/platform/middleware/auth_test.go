package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerAuth(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) })

	for _, tc := range []struct {
		token, header string
		want          int
	}{
		{"s3cret", "Bearer s3cret", http.StatusTeapot},
		{"s3cret", "Bearer wrong", http.StatusUnauthorized},
		{"s3cret", "bearer s3cret", http.StatusUnauthorized},
		{"s3cret", "Basic czNjcmV0", http.StatusUnauthorized},
		{"s3cret", "", http.StatusUnauthorized},
		{"s3cret", "Bearer s3cret ", http.StatusUnauthorized},
		{"", "Bearer ", http.StatusUnauthorized}, // an unset token never authenticates
	} {
		req := httptest.NewRequest("GET", "/api/admin/rounds", nil)
		if tc.header != "" {
			req.Header.Set("Authorization", tc.header)
		}
		rec := httptest.NewRecorder()
		BearerAuth(tc.token)(ok).ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("token=%q header=%q: %d; want %d", tc.token, tc.header, rec.Code, tc.want)
		}
		if tc.want == http.StatusTeapot && rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("admin responses must not be cached")
		}
	}
}

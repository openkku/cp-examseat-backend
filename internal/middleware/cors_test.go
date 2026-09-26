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

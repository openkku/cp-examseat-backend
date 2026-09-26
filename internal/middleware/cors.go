package middleware

import (
	"net/http"
	"slices"
)

// CORS allows cross-origin requests from the given origins ("*" allows any).
// With no origins configured it is a no-op, which suits deployments where the
// frontend proxies /api to the backend on the same origin.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAny := slices.Contains(allowedOrigins, "*")
	return func(next http.Handler) http.Handler {
		if len(allowedOrigins) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowAny || slices.Contains(allowedOrigins, origin)) {
				h := w.Header()
				if allowAny {
					h.Set("Access-Control-Allow-Origin", "*")
				} else {
					h.Set("Access-Control-Allow-Origin", origin)
					h.Add("Vary", "Origin")
				}
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

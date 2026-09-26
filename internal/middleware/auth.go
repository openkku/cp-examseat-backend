package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

// BearerAuth requires "Authorization: Bearer <token>". Tokens are compared
// in constant time (via their hashes, so length is not leaked either). A
// bearer header is never attached by browsers on their own, so these routes
// are not exposed to CSRF.
func BearerAuth(token string) func(http.Handler) http.Handler {
	want := sha256.Sum256([]byte(token))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			gotHash := sha256.Sum256([]byte(got))
			if !ok || token == "" || subtle.ConstantTimeCompare(gotHash[:], want[:]) != 1 {
				w.Header().Set("WWW-Authenticate", `Bearer realm="admin"`)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"Unauthorized"}` + "\n"))
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			next.ServeHTTP(w, r)
		})
	}
}

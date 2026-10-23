package httpx

import (
	"net/http"
	"net/url"
)

const (
	// MaxParamLength bounds every query value. Values become cache keys, so
	// unbounded lengths would let a client exhaust memory.
	MaxParamLength = 64
	// MaxStudentIDDigits bounds a normalized student ID (real IDs have 10 digits).
	MaxStudentIDDigits = 20
)

// ParamsTooLong reports whether any value of the query exceeds MaxParamLength.
func ParamsTooLong(q url.Values) bool {
	for _, values := range q {
		for _, v := range values {
			if len(v) > MaxParamLength {
				return true
			}
		}
	}
	return false
}

// ClientIPFunc identifies the visitor of a request for per-client limits.
type ClientIPFunc func(*http.Request) string

// ClientIP returns the visitor's address, or "" (never limited) without a resolver.
func ClientIP(f ClientIPFunc, r *http.Request) string {
	if f == nil {
		return ""
	}
	return f(r)
}

package controllers

import "net/url"

const (
	// maxParamLength bounds every query value. Values become cache keys, so
	// unbounded lengths would let a client exhaust memory.
	maxParamLength = 64
	// maxStudentIDDigits bounds a normalized student ID (real IDs have 10 digits).
	maxStudentIDDigits = 20
)

// paramsTooLong reports whether any value of the query exceeds maxParamLength.
func paramsTooLong(q url.Values) bool {
	for _, values := range q {
		for _, v := range values {
			if len(v) > maxParamLength {
				return true
			}
		}
	}
	return false
}

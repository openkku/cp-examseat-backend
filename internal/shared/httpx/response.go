// Package httpx holds the HTTP helpers shared by every slice's handlers: JSON
// and error bodies, the cache of rendered, pre-compressed responses, request
// parameter limits and client identification.
package httpx

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the JSON body of every API error.
type ErrorResponse struct {
	Error string `json:"error"`
}

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes a JSON-formatted error response to the client.
func Error(w http.ResponseWriter, message string, status int) {
	JSON(w, status, ErrorResponse{Error: message})
}

// NotFound writes a standard JSON 404 error response when no data is found.
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "No data found"
	}
	Error(w, message, http.StatusNotFound)
}

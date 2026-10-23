// Package views is the "V" of the MVC architecture: it turns model data into
// HTTP representations (JSON payloads, iCalendar feeds, error bodies) and owns
// the caching of rendered, pre-compressed responses.
package views

import (
	"encoding/json"
	"net/http"
	"strconv"
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

// QueuedResponse is the 202 body of a lookup still waiting in the checkex queue.
type QueuedResponse struct {
	Queued     bool `json:"queued"`
	Position   int  `json:"position"`
	RetryAfter int  `json:"retry_after"`
}

// Queued tells the client its lookup is still queued upstream and when to ask again.
func Queued(w http.ResponseWriter, position int, retryAfter int) {
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	JSON(w, http.StatusAccepted, QueuedResponse{Queued: true, Position: position, RetryAfter: retryAfter})
}

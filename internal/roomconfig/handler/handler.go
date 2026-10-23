// Package handler serves room editing (/api/rooms) for the operator app.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/openkku/cp-examseat-backend/internal/room"
	"github.com/openkku/cp-examseat-backend/internal/roomconfig"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

// Config is the room configuration the handler edits.
type Config interface {
	RawMetadata() ([]byte, error)
	SaveRoom(roomID string, meta room.Meta) error
	DeleteRoom(roomID string) error
	Layout(filename string) ([]byte, error)
	SaveLayout(filename string, body []byte) (int, error)
}

// Handler backs the operator app's room editing: room metadata and seating
// layouts on disk.
type Handler struct {
	config Config
}

// New creates a Handler.
func New(config Config) *Handler {
	return &Handler{config: config}
}

var saved = map[string]bool{"success": true}

// Index handles GET /api/rooms, returning metadata.json.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	data, err := h.config.RawMetadata()
	if err != nil {
		httpx.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// SaveRoom handles POST /api/rooms/{roomId}.
func (h *Handler) SaveRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if roomID == "" {
		httpx.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	var meta room.Meta
	if err := json.NewDecoder(r.Body).Decode(&meta); err != nil {
		bodyError(w, err)
		return
	}

	if err := h.config.SaveRoom(roomID, meta); err != nil {
		httpx.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpx.JSON(w, http.StatusOK, saved)
}

// DeleteRoom handles DELETE /api/rooms/{roomId}.
func (h *Handler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if roomID == "" {
		httpx.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	err := h.config.DeleteRoom(roomID)
	switch {
	case errors.Is(err, roomconfig.ErrMetadataNotFound):
		httpx.NotFound(w, "Metadata config file not found")
	case errors.Is(err, roomconfig.ErrRoomNotFound):
		httpx.NotFound(w, "Room not found")
	case err != nil:
		httpx.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		httpx.JSON(w, http.StatusOK, saved)
	}
}

// Layout handles GET /api/rooms/layouts/{filename}.
func (h *Handler) Layout(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		httpx.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	data, err := h.config.Layout(filename)
	if err != nil {
		httpx.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data) // #nosec G705 -- JSON layout file served as application/json with nosniff
}

// SaveLayout handles POST /api/rooms/layouts/{filename}.
func (h *Handler) SaveLayout(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		httpx.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		bodyError(w, err)
		return
	}

	seats, err := h.config.SaveLayout(filename, body)
	if errors.Is(err, roomconfig.ErrInvalidLayout) {
		httpx.Error(w, "Invalid layout JSON", http.StatusBadRequest)
		return
	}
	if err != nil {
		httpx.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"success": true, "seats": seats})
}

// bodyError reports an unreadable request body, with 413 when it exceeded
// the size limit set by middleware.MaxBody.
func bodyError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		httpx.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	httpx.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
}

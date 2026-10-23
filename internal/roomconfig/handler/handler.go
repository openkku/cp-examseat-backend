// Package handler serves the room-config tool's JSON API and its router.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/openkku/cp-examseat-backend/internal/platform/middleware"

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
	SaveLayout(filename string, body []byte) error
}

// Handler backs the room-config tool used to edit room metadata and seating
// layouts on disk.
type Handler struct {
	config Config
}

// New creates a Handler.
func New(config Config) *Handler {
	return &Handler{config: config}
}

var saved = map[string]bool{"success": true}

// Index handles GET /api/config, returning metadata.json.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	data, err := h.config.RawMetadata()
	if err != nil {
		httpx.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// SaveRoom handles POST /api/config/{roomId}.
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

// DeleteRoom handles DELETE /api/config/{roomId}.
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

// Layout handles GET /api/layout/{filename}.
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

// SaveLayout handles POST /api/layout/{filename}.
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

	err = h.config.SaveLayout(filename, body)
	if errors.Is(err, roomconfig.ErrInvalidLayout) {
		httpx.Error(w, "Invalid layout JSON", http.StatusBadRequest)
		return
	}
	if err != nil {
		httpx.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	httpx.JSON(w, http.StatusOK, saved)
}

// bodyError reports an unreadable request body, with 413 when it exceeded
// the size limit set by middleware.LocalToolGuard.
func bodyError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		httpx.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	httpx.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
}

// NewRouter builds the router of the room-config tool: its JSON API plus the
// embedded editor UI served by ui for every other path. The UI is served from
// the same origin, so no CORS is allowed.
func NewRouter(h *Handler, ui http.HandlerFunc) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.LocalToolGuard)

	r.Route("/api", func(r chi.Router) {
		r.Get("/config", h.Index)
		r.Post("/config/{roomId}", h.SaveRoom)
		r.Delete("/config/{roomId}", h.DeleteRoom)
		r.Get("/layout/{filename}", h.Layout)
		r.Post("/layout/{filename}", h.SaveLayout)
	})

	r.Get("/*", ui)

	return r
}

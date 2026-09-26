package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/repositories/filesystem"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// RoomConfigController backs the room-config tool used to edit room metadata
// and seating layouts on disk.
type RoomConfigController struct {
	store *filesystem.RoomConfigStore
}

// NewRoomConfigController creates a RoomConfigController.
func NewRoomConfigController(store *filesystem.RoomConfigStore) *RoomConfigController {
	return &RoomConfigController{store: store}
}

var saved = map[string]bool{"success": true}

// Index handles GET /api/config, returning metadata.json.
func (c *RoomConfigController) Index(w http.ResponseWriter, r *http.Request) {
	data, err := c.store.RawMetadata()
	if err != nil {
		views.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// SaveRoom handles POST /api/config/{roomId}.
func (c *RoomConfigController) SaveRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if roomID == "" {
		views.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	var meta models.RoomMeta
	if err := json.NewDecoder(r.Body).Decode(&meta); err != nil {
		bodyError(w, err)
		return
	}

	if err := c.store.SaveRoom(roomID, meta); err != nil {
		views.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.JSON(w, http.StatusOK, saved)
}

// DeleteRoom handles DELETE /api/config/{roomId}.
func (c *RoomConfigController) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomId")
	if roomID == "" {
		views.Error(w, "Room ID is required", http.StatusBadRequest)
		return
	}

	err := c.store.DeleteRoom(roomID)
	switch {
	case errors.Is(err, filesystem.ErrMetadataNotFound):
		views.NotFound(w, "Metadata config file not found")
	case errors.Is(err, filesystem.ErrRoomNotFound):
		views.NotFound(w, "Room not found")
	case err != nil:
		views.Error(w, err.Error(), http.StatusInternalServerError)
	default:
		views.JSON(w, http.StatusOK, saved)
	}
}

// Layout handles GET /api/layout/{filename}.
func (c *RoomConfigController) Layout(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		views.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	data, err := c.store.Layout(filename)
	if err != nil {
		views.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data) // #nosec G705 -- JSON layout file served as application/json with nosniff
}

// SaveLayout handles POST /api/layout/{filename}.
func (c *RoomConfigController) SaveLayout(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		views.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		bodyError(w, err)
		return
	}

	err = c.store.SaveLayout(filename, body)
	if errors.Is(err, filesystem.ErrInvalidLayout) {
		views.Error(w, "Invalid layout JSON", http.StatusBadRequest)
		return
	}
	if err != nil {
		views.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	views.JSON(w, http.StatusOK, saved)
}

// bodyError reports an unreadable request body, with 413 when it exceeded
// the size limit set by middleware.LocalToolGuard.
func bodyError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		views.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	views.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
}

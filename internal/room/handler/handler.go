// Package handler serves room layouts and room images over HTTP.
package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/openkku/cp-examseat-backend/internal/room"
	"github.com/openkku/cp-examseat-backend/internal/room/presenter"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

// Rooms is the room catalog the handler reads.
type Rooms interface {
	All() map[string]room.Room
	Find(names []string) map[string]room.Room
	ConfiguredCount() int
}

// Handler serves room layouts and room images.
type Handler struct {
	rooms    Rooms
	cache    *httpx.ResponseCache
	imageDir string
}

// New creates a Handler serving local images from imageDir.
func New(rooms Rooms, cache *httpx.ResponseCache, imageDir string) *Handler {
	return &Handler{rooms: rooms, cache: cache, imageDir: imageDir}
}

// Index handles GET /api/room[?room=A&room=B|?room=A,B][&no_layout=true].
// Without a room filter every room is returned.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	noLayout := r.URL.Query().Get("no_layout") == "true"

	var requested []string
	for _, q := range r.URL.Query()["room"] {
		for _, part := range strings.Split(q, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				if len(trimmed) > httpx.MaxParamLength {
					httpx.Error(w, "Parameter too long", http.StatusBadRequest)
					return
				}
				requested = append(requested, trimmed)
			}
		}
	}

	// Asking for more rooms than exist can only be abuse of the cache key space.
	if len(requested) > h.rooms.ConfiguredCount() {
		httpx.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	key := fmt.Sprintf("room:%t", noLayout)
	for _, name := range requested {
		key += ":" + name
	}

	err := h.cache.ServeJSON(w, r, key, func() (int, any, error) {
		var rooms map[string]room.Room
		if len(requested) == 0 {
			rooms = h.rooms.All()
		} else {
			rooms = h.rooms.Find(requested)
		}
		if len(rooms) == 0 {
			return http.StatusNotFound, httpx.ErrorResponse{Error: "Rooms not found"}, nil
		}
		return http.StatusOK, presenter.RoomsPayload(rooms, !noLayout), nil
	})
	if err != nil {
		log.Printf("Room render error: %v", err)
		httpx.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Image handles GET /room/image/*, serving files from the room image directory.
func (h *Handler) Image(w http.ResponseWriter, r *http.Request) {
	// http.Dir confines the lookup to imageDir, rejecting ".." escapes.
	f, err := http.Dir(h.imageDir).Open("/" + chi.URLParam(r, "*"))
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

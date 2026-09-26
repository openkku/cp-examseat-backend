package controllers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/openkku/cp-examseat-backend/internal/models"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// RoomController serves room layouts and room images.
type RoomController struct {
	rooms    *services.RoomService
	cache    *views.ResponseCache
	imageDir string
}

// NewRoomController creates a RoomController serving local images from imageDir.
func NewRoomController(rooms *services.RoomService, cache *views.ResponseCache, imageDir string) *RoomController {
	return &RoomController{rooms: rooms, cache: cache, imageDir: imageDir}
}

// Index handles GET /api/room[?room=A&room=B|?room=A,B][&no_layout=true].
// Without a room filter every room is returned.
func (c *RoomController) Index(w http.ResponseWriter, r *http.Request) {
	noLayout := r.URL.Query().Get("no_layout") == "true"

	var requested []string
	for _, q := range r.URL.Query()["room"] {
		for _, part := range strings.Split(q, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				if len(trimmed) > maxParamLength {
					views.Error(w, "Parameter too long", http.StatusBadRequest)
					return
				}
				requested = append(requested, trimmed)
			}
		}
	}

	// Asking for more rooms than exist can only be abuse of the cache key space.
	if len(requested) > c.rooms.ConfiguredCount() {
		views.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	key := fmt.Sprintf("room:%t", noLayout)
	for _, name := range requested {
		key += ":" + name
	}

	err := c.cache.ServeJSON(w, r, key, func() (int, any, error) {
		var rooms map[string]models.Room
		if len(requested) == 0 {
			rooms = c.rooms.All()
		} else {
			rooms = c.rooms.Find(requested)
		}
		if len(rooms) == 0 {
			return http.StatusNotFound, views.ErrorResponse{Error: "Rooms not found"}, nil
		}
		return http.StatusOK, views.RoomsPayload(rooms, !noLayout), nil
	})
	if err != nil {
		log.Printf("Room render error: %v", err)
		views.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Image handles GET /room/image/*, serving files from the room image directory.
func (c *RoomController) Image(w http.ResponseWriter, r *http.Request) {
	// http.Dir confines the lookup to imageDir, rejecting ".." escapes.
	f, err := http.Dir(c.imageDir).Open("/" + chi.URLParam(r, "*"))
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

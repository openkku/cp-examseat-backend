// Package routes maps URLs to controller actions.
package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/openkku/cp-examseat-backend/internal/controllers"
	"github.com/openkku/cp-examseat-backend/internal/middleware"
)

// Controllers groups every controller mounted by the public API router.
type Controllers struct {
	Exam     *controllers.ExamController
	Explore  *controllers.ExploreController
	Calendar *controllers.CalendarController
	Room     *controllers.RoomController
	Round    *controllers.RoundController
	Stats    *controllers.StatsController
}

// New builds the public API router.
func New(c Controllers, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS(allowedOrigins))
	r.Use(middleware.Compress)

	r.Get("/healthz", controllers.Health)

	r.Route("/api", func(r chi.Router) {
		r.Get("/rounds", c.Round.Index)          // [{id, label}]
		r.Get("/exam", c.Exam.Show)              // ?id=&round=
		r.Get("/calendar/{id}", c.Calendar.Show) // {id} or {id}.ics
		r.Get("/explore", c.Explore.Index)       // ?round=&room=&date=&time=[&seat=]
		r.Get("/options", c.Explore.Options)     // ?type=dates|times|rooms&round=[&date=&time=]
		r.Get("/stats", c.Stats.Index)
		r.Get("/room", c.Room.Index) // [?room=A,B][&no_layout=true]
	})

	r.Get("/room/image/*", c.Room.Image)

	return r
}

// NewRoomConfig builds the router of the room-config tool: its JSON API plus
// the embedded editor UI served by ui for every other path.
func NewRoomConfig(c *controllers.RoomConfigController, ui http.HandlerFunc) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS([]string{"*"}))

	r.Route("/api", func(r chi.Router) {
		r.Get("/config", c.Index)
		r.Post("/config/{roomId}", c.SaveRoom)
		r.Delete("/config/{roomId}", c.DeleteRoom)
		r.Get("/layout/{filename}", c.Layout)
		r.Post("/layout/{filename}", c.SaveLayout)
	})

	r.Get("/*", ui)

	return r
}

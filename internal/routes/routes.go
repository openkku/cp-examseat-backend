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
	// Admin is mounted under /api/admin when set.
	Admin *controllers.AdminController
}

// Options configures cross-cutting behavior of the API router.
type Options struct {
	AllowedOrigins []string
	// RateLimit is applied to every route except /healthz.
	RateLimit func(http.Handler) http.Handler
	// AdminToken protects the admin API.
	AdminToken string
}

// New builds the public API router.
func New(c Controllers, opts Options) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(opts.AllowedOrigins))

	r.Get("/healthz", controllers.Health)

	r.Group(func(r chi.Router) {
		if opts.RateLimit != nil {
			r.Use(opts.RateLimit)
		}
		r.Use(middleware.Compress)
		mountAPI(r, c, opts)
	})

	return r
}

func mountAPI(r chi.Router, c Controllers, opts Options) {
	r.Route("/api", func(r chi.Router) {
		r.Get("/rounds", c.Round.Index)          // [{id, label}]
		r.Get("/exam", c.Exam.Show)              // ?id=&round=
		r.Get("/calendar/{id}", c.Calendar.Show) // {id} or {id}.ics
		r.Get("/explore", c.Explore.Index)       // ?round=&room=&date=&time=[&seat=]
		r.Get("/options", c.Explore.Options)     // ?type=dates|times|rooms&round=[&date=&time=]
		r.Get("/stats", c.Stats.Index)
		r.Get("/room", c.Room.Index) // [?room=A,B][&no_layout=true]

		if c.Admin != nil && opts.AdminToken != "" {
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.BearerAuth(opts.AdminToken))
				r.Get("/status", c.Admin.Status)
				r.Get("/rounds", c.Admin.Rounds)
				r.Patch("/rounds/{round}", c.Admin.RenameRound)
				r.Delete("/rounds/{round}", c.Admin.DeleteRound) // [?custom_id=]
				r.Post("/import", c.Admin.Import)
				r.Post("/reload", c.Admin.Reload)
				r.Post("/backups", c.Admin.CreateBackup)
				r.Get("/backup", c.Admin.DownloadBackup)
			})
		}
	})

	r.Get("/room/image/*", c.Room.Image)
}

// NewRoomConfig builds the router of the room-config tool: its JSON API plus
// the embedded editor UI served by ui for every other path. The UI is served
// from the same origin, so no CORS is allowed.
func NewRoomConfig(c *controllers.RoomConfigController, ui http.HandlerFunc) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.LocalToolGuard)

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

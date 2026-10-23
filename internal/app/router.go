package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	examhandler "github.com/openkku/cp-examseat-backend/internal/exam/handler"
	"github.com/openkku/cp-examseat-backend/internal/platform/middleware"
	roomhandler "github.com/openkku/cp-examseat-backend/internal/room/handler"
	roundhandler "github.com/openkku/cp-examseat-backend/internal/round/handler"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
	statshandler "github.com/openkku/cp-examseat-backend/internal/stats/handler"
)

// handlers groups every handler mounted by the public API router.
type handlers struct {
	Exam     *examhandler.Exam
	Explore  *examhandler.Explore
	Calendar *examhandler.Calendar
	Room     *roomhandler.Handler
	Round    *roundhandler.Handler
	Stats    *statshandler.Handler
}

// routerOptions configures cross-cutting behavior of the API router.
type routerOptions struct {
	AllowedOrigins []string
	// ClientIPs, when set, makes RemoteAddr the visitor's address (behind
	// trusted proxies) before anything logs or limits the request.
	ClientIPs *middleware.ClientIPResolver
	// RateLimit is applied to every route except /healthz.
	RateLimit func(http.Handler) http.Handler
}

// newRouter builds the public API router.
func newRouter(c handlers, opts routerOptions) http.Handler {
	r := chi.NewRouter()

	if opts.ClientIPs != nil {
		r.Use(middleware.ResolveClientAddr(opts.ClientIPs))
	}
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(opts.AllowedOrigins))

	r.Get("/healthz", httpx.Health)

	r.Group(func(r chi.Router) {
		if opts.RateLimit != nil {
			r.Use(opts.RateLimit)
		}
		r.Use(middleware.Compress)
		mountAPI(r, c, opts)
	})

	return r
}

func mountAPI(r chi.Router, c handlers, opts routerOptions) {
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
}

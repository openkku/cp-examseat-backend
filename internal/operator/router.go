// Package operator builds the operator app's web API: room editing and round
// management for the people who run the site. It is served only on loopback
// (through an SSH tunnel on the server) and is never part of the public site.
package operator

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	adminhandler "github.com/openkku/cp-examseat-backend/internal/admin/handler"
	"github.com/openkku/cp-examseat-backend/internal/datacheck"
	"github.com/openkku/cp-examseat-backend/internal/platform/middleware"
	roomconfighandler "github.com/openkku/cp-examseat-backend/internal/roomconfig/handler"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
)

// maxJSONBody bounds every request body except seating-file uploads, which
// the rounds handler caps at MAX_UPLOAD_MB itself.
const maxJSONBody = 1 << 20 // 1 MiB

// HealthChecker executes data health checks.
type HealthChecker interface {
	Run(ctx context.Context) (datacheck.Report, error)
}

// NewRouter serves the rooms and rounds APIs plus the embedded UI (ui, for
// every other path). The UI is served from the same origin, so no CORS is
// allowed.
func NewRouter(rooms *roomconfighandler.Handler, rounds *adminhandler.Handler, preview *PreviewHandler, health HealthChecker, ui http.HandlerFunc) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.LocalToolGuard)

	r.Route("/api", func(r chi.Router) {
		r.Route("/rooms", func(r chi.Router) {
			r.Use(middleware.MaxBody(maxJSONBody))
			r.Get("/", rooms.Index)
			r.Post("/{roomId}", rooms.SaveRoom)
			r.Delete("/{roomId}", rooms.DeleteRoom)
			r.Get("/layouts/{filename}", rooms.Layout)
			r.Post("/layouts/{filename}", rooms.SaveLayout)
		})
		r.Route("/rounds", func(r chi.Router) {
			r.Post("/import", rounds.Import) // multipart; capped by the handler
			r.Post("/import/preview", rounds.PreviewImport)
			r.Group(func(r chi.Router) {
				r.Use(middleware.MaxBody(maxJSONBody))
				r.Get("/", rounds.Rounds)
				r.Patch("/{round}", rounds.RenameRound)
				r.Delete("/{round}", rounds.DeleteRound) // [?custom_id=]
				r.Post("/reload", rounds.Reload)
			})
		})
		r.Route("/preview", func(r chi.Router) {
			r.Use(middleware.MaxBody(maxJSONBody))
			r.Get("/options", preview.Options)
			r.Get("/seats", preview.Seats)
		})
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			report, err := health.Run(r.Context())
			if err != nil {
				httpx.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			httpx.JSON(w, http.StatusOK, report)
		})
	})

	r.Get("/*", ui)

	return r
}

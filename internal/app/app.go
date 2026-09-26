// Package app is the composition root: it wires repositories, services,
// views and controllers into an HTTP handler.
package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/controllers"
	"github.com/openkku/cp-examseat-backend/internal/repositories/filesystem"
	"github.com/openkku/cp-examseat-backend/internal/repositories/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/routes"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// App is a fully wired backend.
type App struct {
	Handler http.Handler
	db      *sqlite.Database
}

// New opens the database, preloads rounds, rooms and analytics, and builds the router.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := sqlite.New(cfg.DatabasePath())
	if err != nil {
		return nil, err
	}

	// Model layer
	rounds := services.NewRoundService(db)
	if err := rounds.Load(ctx); err != nil {
		log.Println("⚠️ Warning: Could not load rounds:", err)
	}

	stats := services.NewStatsService(db)
	if err := stats.Refresh(ctx, rounds.Labels()); err != nil {
		log.Printf("⚠️ Warning: Could not generate stats: %v", err)
	}

	log.Println("🗺️  Pre-loading room layouts...")
	rooms := services.NewRoomService(filesystem.LoadRoomCatalog(cfg.RoomDir(), cfg.ImageBaseURL))
	exams := services.NewExamService(db, rooms)

	// Controllers (rendering through the views package)
	handler := routes.New(routes.Controllers{
		Exam:     controllers.NewExamController(exams),
		Explore:  controllers.NewExploreController(exams, views.NewResponseCache(10_000, 5*time.Minute)),
		Calendar: controllers.NewCalendarController(exams),
		Room:     controllers.NewRoomController(rooms, views.NewResponseCache(10_000, 5*time.Minute), cfg.ImageDir()),
		Round:    controllers.NewRoundController(rounds),
		Stats:    controllers.NewStatsController(stats),
	}, cfg.CORSAllowedOrigins)

	return &App{Handler: handler, db: db}, nil
}

// Close releases the database.
func (a *App) Close() error {
	return a.db.Close()
}

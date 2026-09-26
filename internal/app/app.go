// Package app is the composition root: it wires repositories, services,
// views and controllers into an HTTP handler.
package app

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/controllers"
	"github.com/openkku/cp-examseat-backend/internal/middleware"
	"github.com/openkku/cp-examseat-backend/internal/repositories/filesystem"
	"github.com/openkku/cp-examseat-backend/internal/repositories/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/routes"
	"github.com/openkku/cp-examseat-backend/internal/services"
	"github.com/openkku/cp-examseat-backend/internal/views"
)

// App is a fully wired backend.
type App struct {
	Handler http.Handler

	cfg      config.Config
	db       *sqlite.Database
	rounds   *services.RoundService
	stats    *services.StatsService
	rooms    *services.RoomService
	backups  *services.BackupService
	caches   []*views.ResponseCache
	calendar *controllers.CalendarController

	mu         sync.Mutex
	lastReload time.Time
}

// New opens the database, loads rounds, rooms and analytics, and builds the router.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := sqlite.New(cfg.DatabasePath())
	if err != nil {
		return nil, err
	}

	a := &App{
		cfg:     cfg,
		db:      db,
		rounds:  services.NewRoundService(db),
		stats:   services.NewStatsService(db),
		rooms:   services.NewRoomService(filesystem.LoadRoomCatalog(cfg.RoomDir(), cfg.ImageBaseURL)),
		backups: services.NewBackupService(db, cfg.BackupDir, cfg.BackupKeep),
	}
	a.loadData(ctx)

	exams := services.NewExamService(db, a.rooms)
	exploreCache := views.NewResponseCache(10_000, 5*time.Minute)
	roomCache := views.NewResponseCache(10_000, 5*time.Minute)
	a.caches = []*views.ResponseCache{exploreCache, roomCache}
	a.calendar = controllers.NewCalendarController(exams)

	a.Handler = routes.New(routes.Controllers{
		Exam:     controllers.NewExamController(exams),
		Explore:  controllers.NewExploreController(exams, exploreCache),
		Calendar: a.calendar,
		Room:     controllers.NewRoomController(a.rooms, roomCache, cfg.ImageDir()),
		Round:    controllers.NewRoundController(a.rounds),
		Stats:    controllers.NewStatsController(a.stats),
		Admin:    controllers.NewAdminController(db, services.NewIngestService(db), a.backups, a.rooms, a, cfg.MaxUploadBytes),
	}, routes.Options{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		RateLimit:      middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, middleware.NewClientIPResolver(cfg.TrustedProxies)),
		AdminToken:     cfg.AdminToken,
	})

	if cfg.AdminToken == "" {
		log.Println("ℹ️  ADMIN_TOKEN is not set: the admin API is disabled")
	}
	return a, nil
}

// loadData (re)reads rounds and statistics from the database.
func (a *App) loadData(ctx context.Context) {
	if err := a.rounds.Load(ctx); err != nil {
		log.Println("⚠️ Warning: Could not load rounds:", err)
	}
	if err := a.stats.Refresh(ctx, a.rounds.Labels()); err != nil {
		log.Printf("⚠️ Warning: Could not generate stats: %v", err)
	}
	a.mu.Lock()
	a.lastReload = time.Now().UTC()
	a.mu.Unlock()
}

// Reload re-reads rounds, statistics and room files and clears every
// response cache, so imported data shows up without a restart.
func (a *App) Reload(ctx context.Context) error {
	a.rooms.Replace(filesystem.LoadRoomCatalog(a.cfg.RoomDir(), a.cfg.ImageBaseURL))
	if err := a.rounds.Load(ctx); err != nil {
		return err
	}
	if err := a.stats.Refresh(ctx, a.rounds.Labels()); err != nil {
		return err
	}
	for _, c := range a.caches {
		c.Clear()
	}
	a.calendar.ClearCache()

	a.mu.Lock()
	a.lastReload = time.Now().UTC()
	a.mu.Unlock()
	log.Println("🔄 Data reloaded")
	return nil
}

// LastReload is when data was last (re)loaded.
func (a *App) LastReload() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastReload
}

// RunBackups takes scheduled backups until ctx is cancelled (no-op without BACKUP_DIR).
func (a *App) RunBackups(ctx context.Context) {
	if !a.backups.Enabled() {
		return
	}
	log.Printf("💾 Scheduled backups every %s to %s (keeping %d)", a.cfg.BackupInterval, a.cfg.BackupDir, a.cfg.BackupKeep)
	a.backups.Schedule(ctx, a.cfg.BackupInterval)
}

// Close releases the database.
func (a *App) Close() error {
	return a.db.Close()
}

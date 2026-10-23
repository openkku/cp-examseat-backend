// Package app is the composition root: it opens the database, wires every
// feature slice's core to its adapters and builds the HTTP handler.
package app

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/admin"
	adminhandler "github.com/openkku/cp-examseat-backend/internal/admin/handler"
	"github.com/openkku/cp-examseat-backend/internal/checkex"
	"github.com/openkku/cp-examseat-backend/internal/checkex/scrapler"
	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/database/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/exam"
	examhandler "github.com/openkku/cp-examseat-backend/internal/exam/handler"
	"github.com/openkku/cp-examseat-backend/internal/ingest"
	"github.com/openkku/cp-examseat-backend/internal/ingest/importer"
	"github.com/openkku/cp-examseat-backend/internal/platform/middleware"
	"github.com/openkku/cp-examseat-backend/internal/room"
	roomfs "github.com/openkku/cp-examseat-backend/internal/room/filesystem"
	roomhandler "github.com/openkku/cp-examseat-backend/internal/room/handler"
	"github.com/openkku/cp-examseat-backend/internal/round"
	roundhandler "github.com/openkku/cp-examseat-backend/internal/round/handler"
	"github.com/openkku/cp-examseat-backend/internal/shared/httpx"
	"github.com/openkku/cp-examseat-backend/internal/stats"
	statshandler "github.com/openkku/cp-examseat-backend/internal/stats/handler"
)

// App is a fully wired backend.
type App struct {
	Handler http.Handler

	db      *sqlite.DB
	rounds  *round.Service
	stats   *stats.Service
	rooms   *room.Service
	checkex *checkex.Service
	// clearCaches drops the cached responses after data changed.
	clearCaches []func()

	mu         sync.Mutex
	lastReload time.Time
}

// New opens the database, loads rounds, rooms and analytics, and builds the router.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := sqlite.Open(cfg.DatabasePath())
	if err != nil {
		return nil, err
	}

	a := &App{
		db:     db,
		rounds: round.NewService(db),
		stats:  stats.NewService(db),
		rooms:  room.NewService(roomfs.Loader{RoomDir: cfg.RoomDir(), ImageBaseURL: cfg.ImageBaseURL}),
		// On-demand lookups from checkex for CHECKEX_ROUNDS (nil, i.e. disabled, when unset).
		checkex: checkex.New(checkex.Options{
			Rounds:         cfg.Checkex.Rounds,
			RoundLabels:    cfg.Checkex.RoundLabels,
			IDPrefixes:     cfg.Checkex.IDPrefixes,
			NotFoundTTL:    cfg.Checkex.NotFoundTTL,
			NotFoundLimit:  cfg.Checkex.NotFoundPerIP,
			NotFoundWindow: cfg.Checkex.NotFoundWindow,
		}, db, scrapler.NewClient(cfg.Checkex.ScraplerURL)),
	}
	a.loadData(ctx)

	var upstream exam.Upstream = exam.NoUpstream{}
	if a.checkex != nil {
		upstream = a.checkex
	}
	exams := exam.NewService(db, upstream, a.rooms)
	admins := admin.NewService(db, db, ingest.NewService(db, importer.Registry{}), a.rooms, a)

	clientIPs := middleware.NewClientIPResolver(cfg.TrustedProxies)
	exploreCache := httpx.NewResponseCache(10_000, 5*time.Minute)
	roomCache := httpx.NewResponseCache(10_000, 5*time.Minute)
	calendar := examhandler.NewCalendar(exams, clientIPs.ClientIP)
	a.clearCaches = []func(){exploreCache.Clear, roomCache.Clear, calendar.ClearCache}

	a.Handler = newRouter(handlers{
		Exam:     examhandler.NewExam(exams, clientIPs.ClientIP),
		Explore:  examhandler.NewExplore(exams, exploreCache),
		Calendar: calendar,
		Room:     roomhandler.New(a.rooms, roomCache, cfg.ImageDir()),
		Round:    roundhandler.New(a.rounds),
		Stats:    statshandler.New(a.stats),
		Admin:    adminhandler.New(admins, cfg.MaxUploadBytes),
	}, routerOptions{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		RateLimit:      middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, clientIPs),
		AdminToken:     cfg.AdminToken,
	})

	if a.checkex != nil {
		log.Printf("🔎 checkex lookups enabled for %v via %s", cfg.Checkex.Rounds, cfg.Checkex.ScraplerURL)
	}
	if cfg.AdminToken == "" {
		log.Println("ℹ️  ADMIN_TOKEN is not set: the admin API is disabled")
	}
	return a, nil
}

// loadData (re)reads rounds and statistics from the database.
func (a *App) loadData(ctx context.Context) {
	a.registerCheckexRounds(ctx)
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
	a.rooms.Reload()
	a.registerCheckexRounds(ctx)
	if err := a.rounds.Load(ctx); err != nil {
		return err
	}
	if err := a.stats.Refresh(ctx, a.rounds.Labels()); err != nil {
		return err
	}
	for _, clear := range a.clearCaches {
		clear()
	}

	a.mu.Lock()
	a.lastReload = time.Now().UTC()
	a.mu.Unlock()
	log.Println("🔄 Data reloaded")
	return nil
}

// registerCheckexRounds makes checkex rounds appear in the round list (again,
// after an admin deleted one) before any of their seats are fetched.
func (a *App) registerCheckexRounds(ctx context.Context) {
	if err := a.checkex.RegisterRounds(ctx); err != nil {
		log.Printf("⚠️ Warning: Could not register checkex rounds: %v", err)
	}
}

// LastReload is when data was last (re)loaded.
func (a *App) LastReload() time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastReload
}

// Close releases the database.
func (a *App) Close() error {
	return a.db.Close()
}

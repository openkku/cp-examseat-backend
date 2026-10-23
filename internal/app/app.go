// Package app is the composition root: it opens the database, wires every
// feature slice's core to its adapters and builds the HTTP handler.
package app

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/openkku/cp-examseat-backend/internal/checkex"
	"github.com/openkku/cp-examseat-backend/internal/checkex/scrapler"
	"github.com/openkku/cp-examseat-backend/internal/config"
	"github.com/openkku/cp-examseat-backend/internal/database/sqlite"
	"github.com/openkku/cp-examseat-backend/internal/exam"
	examhandler "github.com/openkku/cp-examseat-backend/internal/exam/handler"
	"github.com/openkku/cp-examseat-backend/internal/platform/middleware"
	"github.com/openkku/cp-examseat-backend/internal/platform/reloadsignal"
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

	// dataDir holds the .reload marker watched by WatchReloads.
	dataDir string
	// reloading serializes reloads from SIGHUP and the .reload marker.
	reloading sync.Mutex
}

// New opens the database, loads rounds, rooms and analytics, and builds the router.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := sqlite.Open(cfg.DatabasePath())
	if err != nil {
		return nil, err
	}

	a := &App{
		db:      db,
		dataDir: cfg.DataDir,
		rounds:  round.NewService(db),
		stats:   stats.NewService(db),
		rooms:   room.NewService(roomfs.Loader{RoomDir: cfg.RoomDir(), ImageBaseURL: cfg.ImageBaseURL}),
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
	}, routerOptions{
		AllowedOrigins: cfg.CORSAllowedOrigins,
		ClientIPs:      clientIPs,
		RateLimit:      middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst, clientIPs),
	})

	if a.checkex != nil {
		log.Printf("🔎 checkex lookups enabled for %v via %s", cfg.Checkex.Rounds, cfg.Checkex.ScraplerURL)
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
}

// Reload re-reads rounds, statistics and room files and clears every
// response cache, so imported data shows up without a restart.
func (a *App) Reload(ctx context.Context) error {
	a.reloading.Lock()
	defer a.reloading.Unlock()

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

	log.Println("🔄 Data reloaded")
	return nil
}

// registerCheckexRounds makes checkex rounds appear in the round list (again,
// after an operator deleted one) before any of their seats are fetched.
func (a *App) registerCheckexRounds(ctx context.Context) {
	if err := a.checkex.RegisterRounds(ctx); err != nil {
		log.Printf("⚠️ Warning: Could not register checkex rounds: %v", err)
	}
}

// WatchReloads reloads whenever a tool touches $DATA_DIR/.reload (see
// platform/reloadsignal), checking every interval until ctx is done.
func (a *App) WatchReloads(ctx context.Context, interval time.Duration) {
	reloadsignal.Watch(ctx, a.dataDir, interval, a.Reload)
}

// Close releases the database.
func (a *App) Close() error {
	return a.db.Close()
}

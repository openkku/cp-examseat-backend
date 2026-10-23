package checkex

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const notFoundPruneAt = 1000

// notFoundLimiter counts upstream "not found" answers per visitor. Real students look up IDs
// that exist; scripts walking through ID ranges mostly hit not-found, so those visitors run
// out of budget while normal use never does. A nil *notFoundLimiter and an empty key never limit.
type notFoundLimiter struct {
	limit  int
	window time.Duration

	mu      sync.Mutex
	entries map[string]*notFoundEntry
}

type notFoundEntry struct {
	lim  *rate.Limiter
	seen time.Time
}

// newNotFoundLimiter allows limit not-founds per window per client; limit <= 0 disables it.
func newNotFoundLimiter(limit int, window time.Duration) *notFoundLimiter {
	if limit <= 0 || window <= 0 {
		return nil
	}
	return &notFoundLimiter{limit: limit, window: window, entries: map[string]*notFoundEntry{}}
}

// blocked reports whether client has used up its not-found budget.
func (l *notFoundLimiter) blocked(client string) bool {
	if l == nil || client == "" {
		return false
	}
	return l.get(client).Tokens() < 1
}

// record spends one not-found from client's budget.
func (l *notFoundLimiter) record(client string) {
	if l == nil || client == "" {
		return
	}
	l.get(client).Allow()
}

func (l *notFoundLimiter) get(client string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.entries) > notFoundPruneAt {
		// An entry idle for a whole window has refilled completely, so dropping it changes nothing.
		for k, e := range l.entries {
			if now.Sub(e.seen) > l.window {
				delete(l.entries, k)
			}
		}
	}
	e, ok := l.entries[client]
	if !ok {
		e = &notFoundEntry{lim: rate.NewLimiter(rate.Every(l.window/time.Duration(l.limit)), l.limit)}
		l.entries[client] = e
	}
	e.seen = now
	return e.lim
}

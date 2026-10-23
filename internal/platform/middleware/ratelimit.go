package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ClientIPResolver finds the real client address of a request. X-Forwarded-For
// is only honoured when the direct peer is a trusted proxy (the Next.js
// frontend), and then the right-most untrusted hop is the client, so a
// client cannot spoof its address by sending the header itself.
type ClientIPResolver struct {
	trusted []*net.IPNet
}

// NewClientIPResolver trusts the given IPs and CIDRs; "loopback" and
// "private" expand to the loopback and private (RFC 1918 / ULA) ranges.
func NewClientIPResolver(trustedProxies []string) *ClientIPResolver {
	r := &ClientIPResolver{}
	for _, entry := range trustedProxies {
		switch strings.ToLower(entry) {
		case "loopback":
			r.add("127.0.0.0/8", "::1/128")
		case "private":
			r.add("10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7")
		default:
			if !strings.Contains(entry, "/") {
				if ip := net.ParseIP(entry); ip != nil && ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			}
			r.add(entry)
		}
	}
	return r
}

func (r *ClientIPResolver) add(cidrs ...string) {
	for _, c := range cidrs {
		if _, n, err := net.ParseCIDR(c); err == nil {
			r.trusted = append(r.trusted, n)
		}
	}
}

func (r *ClientIPResolver) isTrusted(ip net.IP) bool {
	for _, n := range r.trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP returns the client address of req.
func (r *ClientIPResolver) ClientIP(req *http.Request) string {
	peer := hostname(req.RemoteAddr)
	peerIP := net.ParseIP(peer)
	if peerIP == nil || !r.isTrusted(peerIP) {
		return peer
	}

	hops := strings.Split(strings.Join(req.Header.Values("X-Forwarded-For"), ","), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		ip := net.ParseIP(strings.TrimSpace(hops[i]))
		if ip == nil {
			break
		}
		if !r.isTrusted(ip) {
			return ip.String()
		}
		peer = ip.String()
	}
	return peer
}

// RateLimiter applies a token bucket per client IP.
type RateLimiter struct {
	limit       rate.Limit
	burst       int
	maxVisitors int
	resolver    *ClientIPResolver
	now         func() time.Time

	mu        sync.Mutex
	visitors  map[string]*visitor
	lastSweep time.Time
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// idleTTL is how long an idle client's bucket is kept.
const idleTTL = 10 * time.Minute

// maxVisitors caps the number of tracked clients. Past it, new clients share
// one overflow bucket, so a flood of distinct (possibly spoofed) addresses
// cannot grow memory without bound.
const maxVisitors = 100_000

const overflowKey = "\x00overflow"

// NewRateLimiter allows rps requests per second with the given burst per client.
func NewRateLimiter(rps float64, burst int, resolver *ClientIPResolver) *RateLimiter {
	return &RateLimiter{
		limit:       rate.Limit(rps),
		burst:       burst,
		maxVisitors: maxVisitors,
		resolver:    resolver,
		now:         time.Now,
		visitors:    make(map[string]*visitor),
	}
}

func (rl *RateLimiter) reserve(key string) (ok bool, retryAfter time.Duration) {
	now := rl.now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Drop idle buckets so memory stays bounded by active clients.
	if now.Sub(rl.lastSweep) > time.Minute {
		for k, v := range rl.visitors {
			if now.Sub(v.lastSeen) > idleTTL {
				delete(rl.visitors, k)
			}
		}
		rl.lastSweep = now
	}

	v, found := rl.visitors[key]
	if !found && len(rl.visitors) >= rl.maxVisitors {
		key = overflowKey
		v, found = rl.visitors[key]
	}
	if !found {
		v = &visitor{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[key] = v
	}
	v.lastSeen = now

	res := v.limiter.ReserveN(now, 1)
	if delay := res.DelayFrom(now); delay > 0 {
		res.CancelAt(now)
		return false, delay
	}
	return true, 0
}

// Middleware rejects requests over the limit with 429 and a Retry-After header.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, retryAfter := rl.reserve(rl.resolver.ClientIP(r)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Too many requests, please slow down"}` + "\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RateLimit returns rate limiting middleware, or a pass-through when rps is 0.
func RateLimit(rps float64, burst int, resolver *ClientIPResolver) func(http.Handler) http.Handler {
	if rps <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if burst < 1 {
		burst = 1
	}
	return NewRateLimiter(rps, burst, resolver).Middleware
}

// Package config loads runtime settings from environment variables.
package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds every runtime setting of the backend.
type Config struct {
	// Port is the TCP port the HTTP server listens on (PORT, default 8080).
	Port string
	// DataDir is the directory holding exams.db and room assets (DATA_DIR, default "data").
	DataDir string
	// ImageBaseURL is an optional CDN prefix for room images (IMAGE_BASE_URL).
	ImageBaseURL string
	// CORSAllowedOrigins lists origins allowed to call the API cross-origin
	// (CORS_ALLOWED_ORIGINS, comma-separated, "*" for any). Empty disables CORS headers.
	CORSAllowedOrigins []string

	// RateLimitRPS is the sustained request rate allowed per client IP
	// (RATE_LIMIT_RPS, default 10; 0 disables rate limiting).
	RateLimitRPS float64
	// RateLimitBurst is the burst size per client IP (RATE_LIMIT_BURST, default 40).
	RateLimitBurst int
	// TrustedProxies lists IPs/CIDRs whose X-Forwarded-For header is trusted
	// to identify the client (TRUSTED_PROXIES). "loopback" and "private" are
	// shorthands; default "loopback,private" suits a backend behind the frontend.
	TrustedProxies []string

	// MaxUploadBytes bounds operator app imports (MAX_UPLOAD_MB, default 20).
	MaxUploadBytes int64

	// Checkex configures on-demand seat lookups from checkex through scrapler.
	Checkex Checkex
}

// Checkex configures on-demand seat lookups from checkex.computing.kku.ac.th
// through the scrapler service (see docs/env.md).
type Checkex struct {
	// ScraplerURL is the scrapler base URL (SCRAPLER_URL, default http://127.0.0.1:8090).
	ScraplerURL string
	// Rounds are the round IDs served by checkex (CHECKEX_ROUNDS, comma-separated).
	// Empty disables the feature.
	Rounds []string
	// RoundLabels overrides derived round labels (CHECKEX_ROUND_LABELS, "id=label;id=label").
	RoundLabels map[string]string
	// IDPrefixes are the intake years sent upstream (CHECKEX_ID_PREFIXES, default "68,69").
	IDPrefixes []string
	// NotFoundTTL is how long a "not found" answer is trusted (CHECKEX_NOTFOUND_TTL, default 6h).
	NotFoundTTL time.Duration
	// NotFoundPerIP is the not-found answers one visitor may cause per window
	// (CHECKEX_NOTFOUND_PER_IP, default 5; 0 disables the limit).
	NotFoundPerIP int
	// NotFoundWindow is the window of NotFoundPerIP (CHECKEX_NOTFOUND_WINDOW, default 1h).
	NotFoundWindow time.Duration
}

// Load reads the configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:               getEnv("PORT", "8080"),
		DataDir:            GetDataDir(),
		ImageBaseURL:       strings.TrimSuffix(os.Getenv("IMAGE_BASE_URL"), "/"),
		CORSAllowedOrigins: splitList(os.Getenv("CORS_ALLOWED_ORIGINS")),
		RateLimitRPS:       getFloat("RATE_LIMIT_RPS", 10),
		RateLimitBurst:     getInt("RATE_LIMIT_BURST", 40),
		TrustedProxies:     splitList(getEnv("TRUSTED_PROXIES", "loopback,private")),
		MaxUploadBytes:     int64(getInt("MAX_UPLOAD_MB", 20)) << 20,
		Checkex: Checkex{
			ScraplerURL: strings.TrimRight(getEnv("SCRAPLER_URL", "http://127.0.0.1:8090"), "/"),
			Rounds:      splitList(os.Getenv("CHECKEX_ROUNDS")),
			RoundLabels: parseLabels(os.Getenv("CHECKEX_ROUND_LABELS")),
			// TODO(2570): bump intake prefixes to "69,70" for the next academic year.
			IDPrefixes:     splitList(getEnv("CHECKEX_ID_PREFIXES", "68,69")),
			NotFoundTTL:    getDuration("CHECKEX_NOTFOUND_TTL", 6*time.Hour),
			NotFoundPerIP:  getInt("CHECKEX_NOTFOUND_PER_IP", 5),
			NotFoundWindow: getDuration("CHECKEX_NOTFOUND_WINDOW", time.Hour),
		},
	}
}

// parseLabels parses "final_1_2569=ปลายภาค 1/2569;mid_2_2569=...".
func parseLabels(raw string) map[string]string {
	labels := map[string]string{}
	for _, pair := range strings.Split(raw, ";") {
		if k, v, ok := strings.Cut(pair, "="); ok && strings.TrimSpace(k) != "" {
			labels[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return labels
}

// DatabasePath is the location of the SQLite database file.
func (c Config) DatabasePath() string {
	return filepath.Join(c.DataDir, "exams.db")
}

// RoomDir is the directory holding metadata.json, map/ and image/.
func (c Config) RoomDir() string {
	return filepath.Join(c.DataDir, "room")
}

// ImageDir is the directory room images are served from when IMAGE_BASE_URL is unset.
func (c Config) ImageDir() string {
	return filepath.Join(c.RoomDir(), "image")
}

// GetDataDir returns the database and assets directory path from the environment.
// Defaults to "data" if not set.
func GetDataDir() string {
	return getEnv("DATA_DIR", "data")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		log.Printf("⚠️ Invalid %s=%q, using %d", key, raw, fallback)
		return fallback
	}
	return v
}

func getFloat(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		log.Printf("⚠️ Invalid %s=%q, using %g", key, raw, fallback)
		return fallback
	}
	return v
}

func getDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil || v <= 0 {
		log.Printf("⚠️ Invalid %s=%q, using %s", key, raw, fallback)
		return fallback
	}
	return v
}

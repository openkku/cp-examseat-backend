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

	// AdminToken enables the /api/admin API when set (ADMIN_TOKEN).
	AdminToken string
	// MaxUploadBytes bounds admin import uploads (MAX_UPLOAD_MB, default 20).
	MaxUploadBytes int64

	// BackupDir enables scheduled SQLite backups when set (BACKUP_DIR).
	BackupDir string
	// BackupInterval is the time between scheduled backups (BACKUP_INTERVAL, default 24h).
	BackupInterval time.Duration
	// BackupKeep is the number of scheduled backups kept (BACKUP_KEEP, default 7).
	BackupKeep int
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
		AdminToken:         os.Getenv("ADMIN_TOKEN"),
		MaxUploadBytes:     int64(getInt("MAX_UPLOAD_MB", 20)) << 20,
		BackupDir:          os.Getenv("BACKUP_DIR"),
		BackupInterval:     getDuration("BACKUP_INTERVAL", 24*time.Hour),
		BackupKeep:         getInt("BACKUP_KEEP", 7),
	}
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

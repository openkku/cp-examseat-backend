// Package config loads runtime settings from environment variables.
package config

import (
	"os"
	"path/filepath"
	"strings"
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
}

// Load reads the configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:               getEnv("PORT", "8080"),
		DataDir:            GetDataDir(),
		ImageBaseURL:       strings.TrimSuffix(os.Getenv("IMAGE_BASE_URL"), "/"),
		CORSAllowedOrigins: splitList(os.Getenv("CORS_ALLOWED_ORIGINS")),
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

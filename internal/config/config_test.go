package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"PORT", "DATA_DIR", "RATE_LIMIT_RPS", "RATE_LIMIT_BURST", "TRUSTED_PROXIES", "ADMIN_TOKEN", "MAX_UPLOAD_MB", "BACKUP_DIR", "BACKUP_INTERVAL", "BACKUP_KEEP"} {
		t.Setenv(k, "")
	}
	c := Load()
	if c.Port != "8080" || c.DataDir != "data" || c.RateLimitRPS != 10 || c.RateLimitBurst != 40 {
		t.Errorf("defaults = %+v", c)
	}
	if len(c.TrustedProxies) != 2 || c.AdminToken != "" || c.MaxUploadBytes != 20<<20 {
		t.Errorf("defaults = %+v", c)
	}
	if c.BackupDir != "" || c.BackupInterval != 24*time.Hour || c.BackupKeep != 7 {
		t.Errorf("backup defaults = %+v", c)
	}
}

func TestLoadOverridesAndInvalidValues(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "0")
	t.Setenv("RATE_LIMIT_BURST", "-3")
	t.Setenv("BACKUP_INTERVAL", "6h")
	t.Setenv("BACKUP_KEEP", "nope")
	t.Setenv("TRUSTED_PROXIES", " 10.0.0.1 , 192.168.0.0/16 ")
	c := Load()
	if c.RateLimitRPS != 0 {
		t.Errorf("RATE_LIMIT_RPS=0 must disable limiting, got %v", c.RateLimitRPS)
	}
	if c.RateLimitBurst != 40 || c.BackupKeep != 7 {
		t.Errorf("invalid values must fall back: %+v", c)
	}
	if c.BackupInterval != 6*time.Hour || len(c.TrustedProxies) != 2 || c.TrustedProxies[1] != "192.168.0.0/16" {
		t.Errorf("overrides = %+v", c)
	}
}

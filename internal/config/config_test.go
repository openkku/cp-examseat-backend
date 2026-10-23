package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"PORT", "DATA_DIR", "RATE_LIMIT_RPS", "RATE_LIMIT_BURST", "TRUSTED_PROXIES", "ADMIN_TOKEN", "MAX_UPLOAD_MB"} {
		t.Setenv(k, "")
	}
	c := Load()
	if c.Port != "8080" || c.DataDir != "data" || c.RateLimitRPS != 10 || c.RateLimitBurst != 40 {
		t.Errorf("defaults = %+v", c)
	}
	if len(c.TrustedProxies) != 2 || c.AdminToken != "" || c.MaxUploadBytes != 20<<20 {
		t.Errorf("defaults = %+v", c)
	}
}

func TestLoadOverridesAndInvalidValues(t *testing.T) {
	t.Setenv("RATE_LIMIT_RPS", "0")
	t.Setenv("RATE_LIMIT_BURST", "-3")
	t.Setenv("CHECKEX_NOTFOUND_WINDOW", "6h")
	t.Setenv("MAX_UPLOAD_MB", "nope")
	t.Setenv("TRUSTED_PROXIES", " 10.0.0.1 , 192.168.0.0/16 ")
	c := Load()
	if c.RateLimitRPS != 0 {
		t.Errorf("RATE_LIMIT_RPS=0 must disable limiting, got %v", c.RateLimitRPS)
	}
	if c.RateLimitBurst != 40 || c.MaxUploadBytes != 20<<20 {
		t.Errorf("invalid values must fall back: %+v", c)
	}
	if c.Checkex.NotFoundWindow != 6*time.Hour || len(c.TrustedProxies) != 2 || c.TrustedProxies[1] != "192.168.0.0/16" {
		t.Errorf("overrides = %+v", c)
	}
}

func TestLoadCheckex(t *testing.T) {
	for _, k := range []string{"SCRAPLER_URL", "CHECKEX_ROUNDS", "CHECKEX_ROUND_LABELS", "CHECKEX_ID_PREFIXES", "CHECKEX_NOTFOUND_TTL", "CHECKEX_NOTFOUND_PER_IP", "CHECKEX_NOTFOUND_WINDOW"} {
		t.Setenv(k, "")
	}
	c := Load().Checkex
	if c.ScraplerURL != "http://127.0.0.1:8090" || len(c.Rounds) != 0 || len(c.IDPrefixes) != 2 {
		t.Errorf("defaults = %+v", c)
	}
	if c.NotFoundTTL != 6*time.Hour || c.NotFoundPerIP != 5 || c.NotFoundWindow != time.Hour {
		t.Errorf("defaults = %+v", c)
	}

	t.Setenv("SCRAPLER_URL", "http://scrapler:8090/")
	t.Setenv("CHECKEX_ROUNDS", "final_1_2569, mid_2_2569")
	t.Setenv("CHECKEX_ROUND_LABELS", "final_1_2569=ปลายภาค 1/2569; bad ;=x")
	t.Setenv("CHECKEX_NOTFOUND_PER_IP", "0")
	c = Load().Checkex
	if c.ScraplerURL != "http://scrapler:8090" || len(c.Rounds) != 2 || c.Rounds[1] != "mid_2_2569" {
		t.Errorf("overrides = %+v", c)
	}
	if len(c.RoundLabels) != 1 || c.RoundLabels["final_1_2569"] != "ปลายภาค 1/2569" || c.NotFoundPerIP != 0 {
		t.Errorf("overrides = %+v", c)
	}
}

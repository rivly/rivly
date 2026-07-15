package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("RIVLY_ADDR", "")
	t.Setenv("RIVLY_POLL_INTERVAL", "")
	t.Setenv("RIVLY_TRUSTED_ORIGINS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Addr != ":8080" || cfg.PollInterval != 5*time.Second {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.TrustedOrigins != nil {
		t.Fatalf("expected no trusted origins by default, got %v", cfg.TrustedOrigins)
	}
}

func TestLoadRejectsInvalidTrustedOrigin(t *testing.T) {
	for _, origin := range []string{
		"example.com",
		"https://example.com/",
		"https://",
		"https://example.com/path",
	} {
		t.Run(origin, func(t *testing.T) {
			t.Setenv("RIVLY_TRUSTED_ORIGINS", origin)
			if _, err := Load(); err == nil {
				t.Fatalf("%q must be rejected, not silently ignored", origin)
			}
		})
	}
}

func TestLoadAcceptsValidTrustedOrigins(t *testing.T) {
	t.Setenv("RIVLY_TRUSTED_ORIGINS", "https://rivly.dev, http://localhost:5173")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.TrustedOrigins) != 2 {
		t.Fatalf("expected 2 trusted origins, got %v", cfg.TrustedOrigins)
	}
}

func TestLoadRejectsInvalidPollInterval(t *testing.T) {
	t.Setenv("RIVLY_POLL_INTERVAL", "30x")
	if _, err := Load(); err == nil {
		t.Fatal("a malformed duration must be rejected, not silently replaced by the default")
	}
}

func TestLoadAcceptsValidPollInterval(t *testing.T) {
	t.Setenv("RIVLY_POLL_INTERVAL", "30s")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Fatalf("PollInterval = %v, want 30s", cfg.PollInterval)
	}
}

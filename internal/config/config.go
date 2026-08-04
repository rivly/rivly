package config

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr           string
	DatabasePath   string
	TrustedOrigins []string
	DockerHost     string
	PollInterval   time.Duration
	DataDir        string
	ComposeBin     string
	SetupToken     string
	LogLevel       slog.Level
}

func Load() (Config, error) {
	pollInterval, err := envDuration("RIVLY_POLL_INTERVAL", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	logLevel, err := envLevel("RIVLY_LOG_LEVEL", slog.LevelInfo)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Addr:           env("RIVLY_ADDR", ":8080"),
		DatabasePath:   env("RIVLY_DATABASE", "rivly.db"),
		TrustedOrigins: splitNonEmpty(os.Getenv("RIVLY_TRUSTED_ORIGINS")),
		DockerHost:     env("DOCKER_HOST", "unix:///var/run/docker.sock"),
		PollInterval:   pollInterval,
		DataDir:        env("RIVLY_DATA", "data"),
		ComposeBin:     os.Getenv("RIVLY_COMPOSE_BIN"),
		SetupToken:     os.Getenv("RIVLY_SETUP_TOKEN"),
		LogLevel:       logLevel,
	}

	if err := validateTrustedOrigins(cfg.TrustedOrigins); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validateTrustedOrigins(origins []string) error {
	protection := http.NewCrossOriginProtection()
	for _, origin := range origins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			return fmt.Errorf("RIVLY_TRUSTED_ORIGINS: %q is not a valid origin, expected scheme://host: %w", origin, err)
		}
	}
	return nil
}

func envLevel(key string, fallback slog.Level) (slog.Level, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(v)); err != nil {
		return 0, fmt.Errorf("%s: %q is not a valid log level, use debug, info, warn or error: %w", key, v, err)
	}
	return level, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a valid duration: %w", key, v, err)
	}
	return d, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

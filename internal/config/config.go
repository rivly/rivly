package config

import (
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
}

func Load() Config {
	return Config{
		Addr:           env("RIVLY_ADDR", ":8080"),
		DatabasePath:   env("RIVLY_DATABASE", "rivly.db"),
		TrustedOrigins: splitNonEmpty(os.Getenv("RIVLY_TRUSTED_ORIGINS")),
		DockerHost:     env("DOCKER_HOST", "unix:///var/run/docker.sock"),
		PollInterval:   envDuration("RIVLY_POLL_INTERVAL", 5*time.Second),
		DataDir:        env("RIVLY_DATA", "data"),
		ComposeBin:     env("RIVLY_COMPOSE_BIN", "docker-compose"),
	}
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
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

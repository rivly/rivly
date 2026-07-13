package config

import (
	"os"
	"strings"
)

type Config struct {
	Addr           string
	DatabasePath   string
	TrustedOrigins []string
	DockerHost     string
}

func Load() Config {
	return Config{
		Addr:           env("RIVLY_ADDR", ":8080"),
		DatabasePath:   env("RIVLY_DATABASE", "rivly.db"),
		TrustedOrigins: splitNonEmpty(os.Getenv("RIVLY_TRUSTED_ORIGINS")),
		DockerHost:     env("DOCKER_HOST", "unix:///var/run/docker.sock"),
	}
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

package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/secret"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	sqlDB, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(sqlDB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	cipher, err := secret.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	return NewStore(db.New(sqlDB), cipher)
}

func TestAuthForRoundTrip(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	if _, err := s.Create(ctx, "GitHub", "ghcr.io", "bob", "s3cret-token", "Admin"); err != nil {
		t.Fatalf("create: %v", err)
	}

	auth := s.AuthFor(ctx, "ghcr.io/acme/app:1.2.3")
	if auth == "" {
		t.Fatal("expected auth for ghcr.io image, got empty")
	}
	raw, err := base64.URLEncoding.DecodeString(auth)
	if err != nil {
		t.Fatalf("decode auth: %v", err)
	}
	var cfg struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		ServerAddress string `json:"serveraddress"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Username != "bob" || cfg.Password != "s3cret-token" || cfg.ServerAddress != "ghcr.io" {
		t.Fatalf("auth config: got %+v", cfg)
	}

	if got := s.AuthFor(ctx, "nginx:latest"); got != "" {
		t.Fatalf("expected no auth for docker hub image, got %q", got)
	}
}

func TestUpdateKeepsPasswordWhenEmpty(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	reg, err := s.Create(ctx, "GitHub", "ghcr.io", "bob", "orig-token", "Admin")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := s.Update(ctx, reg.ID, "GitHub Renamed", "ghcr.io", "alice", ""); err != nil {
		t.Fatalf("update: %v", err)
	}

	auth := s.AuthFor(ctx, "ghcr.io/acme/app")
	raw, _ := base64.URLEncoding.DecodeString(auth)
	var cfg struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg.Username != "alice" || cfg.Password != "orig-token" {
		t.Fatalf("after update: got username=%q password=%q", cfg.Username, cfg.Password)
	}
}

func TestDomain(t *testing.T) {
	cases := map[string]string{
		"ghcr.io/acme/app:1.0": "ghcr.io",
		"nginx":                "docker.io",
		"nginx:latest":         "docker.io",
		"registry.example.com:5000/team/img": "registry.example.com:5000",
	}
	for ref, want := range cases {
		if got := Domain(ref); got != want {
			t.Errorf("Domain(%q) = %q, want %q", ref, got, want)
		}
	}
}

package gitcred

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestCredentialsRoundTrip(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, "GitHub", "bob", "ghp_secret-token", "Admin")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	username, token, err := s.Credentials(ctx, created.ID)
	if err != nil {
		t.Fatalf("credentials: %v", err)
	}
	if username != "bob" || token != "ghp_secret-token" {
		t.Fatalf("credentials: got username=%q token=%q", username, token)
	}
}

func TestUpdateKeepsTokenWhenEmpty(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	created, err := s.Create(ctx, "GitHub", "bob", "orig-token", "Admin")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := s.Update(ctx, created.ID, "GitHub Renamed", "alice", ""); err != nil {
		t.Fatalf("update: %v", err)
	}

	username, token, err := s.Credentials(ctx, created.ID)
	if err != nil {
		t.Fatalf("credentials: %v", err)
	}
	if username != "alice" || token != "orig-token" {
		t.Fatalf("after update: got username=%q token=%q", username, token)
	}
}

func TestTestAccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/acme/app/info/refs" || r.URL.Query().Get("service") != "git-upload-pack" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		user, pass, _ := r.BasicAuth()
		if user == "bob" && pass == "good-token" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if err := TestAccess(context.Background(), srv.URL+"/acme/app", "bob", "good-token"); err != nil {
		t.Fatalf("good creds: %v", err)
	}
	if err := TestAccess(context.Background(), srv.URL+"/acme/app", "bob", "bad-token"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("bad creds: want ErrUnauthorized, got %v", err)
	}
	if err := TestAccess(context.Background(), "ftp://nope", "bob", "x"); err == nil {
		t.Fatal("bad scheme: want error, got nil")
	}
}

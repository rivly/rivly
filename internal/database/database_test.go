package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rivly/rivly/internal/database/db"
)

func TestUserRoundTrip(t *testing.T) {
	sqlDB, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := Migrate(sqlDB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	q := db.New(sqlDB)
	ctx := context.Background()

	count, err := q.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 users, got %d", count)
	}

	created, err := q.CreateUser(ctx, db.CreateUserParams{
		Email:       "admin@rivly.dev",
		DisplayName: "Admin",
		Role:        "admin",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if created.ID == 0 || created.CreatedAt == 0 {
		t.Fatalf("expected populated id and created_at, got %+v", created)
	}

	got, err := q.GetUserByEmail(ctx, "admin@rivly.dev")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.ID != created.ID || got.Email != "admin@rivly.dev" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

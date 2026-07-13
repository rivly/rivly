package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
)

func TestLocalRegisterAndAuthenticate(t *testing.T) {
	sqlDB, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()
	if err := database.Migrate(sqlDB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	local := NewLocal(db.New(sqlDB))
	ctx := context.Background()

	if _, err := local.Register(ctx, "Admin@Rivly.dev", "s3cret-password", "Admin", "admin"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, err := local.Authenticate(ctx, "admin@rivly.dev", "s3cret-password")
	if err != nil {
		t.Fatalf("Authenticate (valid): %v", err)
	}
	if user.Email != "admin@rivly.dev" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}

	if _, err := local.Authenticate(ctx, "admin@rivly.dev", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: expected ErrInvalidCredentials, got %v", err)
	}
	if _, err := local.Authenticate(ctx, "nobody@rivly.dev", "whatever"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email: expected ErrInvalidCredentials, got %v", err)
	}
}

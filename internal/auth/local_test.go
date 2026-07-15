package auth

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(sqlDB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return sqlDB
}

func TestLocalRegisterAndAuthenticate(t *testing.T) {
	sqlDB := openTestDB(t)

	local := NewLocal(db.New(sqlDB), sqlDB)
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

func TestLocalRegisterLeavesNoUserWhenCredentialFails(t *testing.T) {
	sqlDB := openTestDB(t)
	if _, err := sqlDB.Exec("DROP TABLE credentials"); err != nil {
		t.Fatalf("drop credentials: %v", err)
	}

	queries := db.New(sqlDB)
	local := NewLocal(queries, sqlDB)
	ctx := context.Background()

	if _, err := local.Register(ctx, "admin@rivly.dev", "s3cret-password", "Admin", "admin"); err == nil {
		t.Fatal("Register: expected an error when the credential insert fails")
	}

	count, err := queries.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers: %v", err)
	}
	if count != 0 {
		t.Fatalf("a half-registered user would lock setup out forever, got %d users", count)
	}
}

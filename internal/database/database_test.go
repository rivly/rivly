package database

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rivly/rivly/internal/database/db"
)

func TestConcurrentWritesNeverLock(t *testing.T) {
	sqlDB, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()
	if err := Migrate(sqlDB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if got := sqlDB.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("sqlite must serialize on a single connection, got %d", got)
	}

	q := db.New(sqlDB)
	ctx := context.Background()
	errs := make(chan error, 32)
	var wg sync.WaitGroup

	for i := range 16 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := q.CreateUser(ctx, db.CreateUserParams{
				Email:       fmt.Sprintf("user%d@rivly.dev", i),
				DisplayName: "User",
				Role:        "admin",
			})
			errs <- err
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx, err := sqlDB.BeginTx(ctx, nil)
			if err != nil {
				errs <- err
				return
			}
			_, err = q.WithTx(tx).CountUsers(ctx)
			if err != nil {
				_ = tx.Rollback()
				errs <- err
				return
			}
			errs <- tx.Commit()
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent access must not fail: %v", err)
		}
	}
}

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

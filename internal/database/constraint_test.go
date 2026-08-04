package database

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
)

func TestIsUniqueViolation(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	const insert = `INSERT INTO users (email, display_name, role) VALUES (?, 'Admin', 'admin')`
	if _, err := db.Exec(insert, "admin@rivly.dev"); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, dup := db.Exec(insert, "admin@rivly.dev")
	if dup == nil {
		t.Fatal("the unique index on users.email is not enforced")
	}
	if !IsUniqueViolation(dup) {
		t.Fatalf("IsUniqueViolation must recognise a duplicate email, got %v", dup)
	}
	if !IsUniqueViolation(fmt.Errorf("create user: %w", dup)) {
		t.Error("IsUniqueViolation must see through a wrapped error")
	}
}

func TestIsUniqueViolationIgnoresOtherErrors(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if IsUniqueViolation(nil) {
		t.Error("nil is not a unique violation")
	}
	if IsUniqueViolation(errors.New("UNIQUE constraint failed")) {
		t.Error("a plain error whose text mentions UNIQUE must not match, that is the bug this replaces")
	}

	_, notNull := db.Exec(`INSERT INTO users (email, display_name, role) VALUES (NULL, 'x', 'admin')`)
	if notNull == nil {
		t.Fatal("expected a NOT NULL violation")
	}
	if IsUniqueViolation(notNull) {
		t.Errorf("a NOT NULL violation is not a unique violation: %v", notNull)
	}
}

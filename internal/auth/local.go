package auth

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rivly/rivly/internal/database/db"
)

var _ Authenticator = (*Local)(nil)

var decoyHash = mustHashDecoy()

func mustHashDecoy() string {
	hash, err := HashPassword("rivly-decoy-value-for-constant-time-auth")
	if err != nil {
		panic(err)
	}
	return hash
}

type Local struct {
	store    Store
	beginner Beginner
}

func NewLocal(store Store, beginner Beginner) *Local {
	return &Local{store: store, beginner: beginner}
}

func (l *Local) Register(ctx context.Context, email, password, displayName, role string) (db.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return db.User{}, err
	}

	tx, err := l.beginner.BeginTx(ctx, nil)
	if err != nil {
		return db.User{}, fmt.Errorf("begin registration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	queries := l.store.WithTx(tx)
	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:       normalizeEmail(email),
		DisplayName: displayName,
		Role:        role,
	})
	if err != nil {
		return db.User{}, fmt.Errorf("create user: %w", err)
	}
	if _, err := queries.CreatePasswordCredential(ctx, db.CreatePasswordCredentialParams{
		UserID: user.ID,
		Secret: sql.NullString{String: hash, Valid: true},
	}); err != nil {
		return db.User{}, fmt.Errorf("create password credential: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return db.User{}, fmt.Errorf("commit registration: %w", err)
	}
	return user, nil
}

func (l *Local) Authenticate(ctx context.Context, email, password string) (db.User, error) {
	user, err := l.store.GetUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		_, _ = VerifyPassword(password, decoyHash)
		return db.User{}, ErrInvalidCredentials
	}
	cred, err := l.store.GetPasswordCredential(ctx, user.ID)
	if err != nil || !cred.Secret.Valid {
		_, _ = VerifyPassword(password, decoyHash)
		return db.User{}, ErrInvalidCredentials
	}
	match, err := VerifyPassword(password, cred.Secret.String)
	if err != nil || !match {
		return db.User{}, ErrInvalidCredentials
	}
	return user, nil
}

package auth

import (
	"context"
	"database/sql"

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
	store Store
}

func NewLocal(store Store) *Local {
	return &Local{store: store}
}

func (l *Local) Register(ctx context.Context, email, password, displayName, role string) (db.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return db.User{}, err
	}
	user, err := l.store.CreateUser(ctx, db.CreateUserParams{
		Email:       normalizeEmail(email),
		DisplayName: displayName,
		Role:        role,
	})
	if err != nil {
		return db.User{}, err
	}
	_, err = l.store.CreatePasswordCredential(ctx, db.CreatePasswordCredentialParams{
		UserID: user.ID,
		Secret: sql.NullString{String: hash, Valid: true},
	})
	if err != nil {
		return db.User{}, err
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

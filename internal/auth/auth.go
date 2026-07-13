package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/rivly/rivly/internal/database/db"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type Store interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	CreatePasswordCredential(ctx context.Context, arg db.CreatePasswordCredentialParams) (db.Credential, error)
	GetPasswordCredential(ctx context.Context, userID int64) (db.Credential, error)
}

type Authenticator interface {
	Authenticate(ctx context.Context, email, password string) (db.User, error)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

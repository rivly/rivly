package auth

import (
	"context"
	"database/sql"
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
	WithTx(tx *sql.Tx) *db.Queries
}

type Beginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type Authenticator interface {
	Authenticate(ctx context.Context, email, password string) (db.User, error)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

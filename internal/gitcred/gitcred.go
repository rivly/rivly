package gitcred

import (
	"context"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/secret"
)

type Credential struct {
	ID        int64
	Name      string
	Username  string
	CreatedBy string
	CreatedAt int64
	UpdatedAt int64
}

type Store struct {
	queries *db.Queries
	cipher  *secret.Cipher
}

func NewStore(queries *db.Queries, cipher *secret.Cipher) *Store {
	return &Store{queries: queries, cipher: cipher}
}

func toCredential(c db.GitCredential) Credential {
	return Credential{
		ID:        c.ID,
		Name:      c.Name,
		Username:  c.Username,
		CreatedBy: c.CreatedBy,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (s *Store) List(ctx context.Context) ([]Credential, error) {
	rows, err := s.queries.ListGitCredentials(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Credential, 0, len(rows))
	for _, c := range rows {
		out = append(out, toCredential(c))
	}
	return out, nil
}

func (s *Store) Create(ctx context.Context, name, username, token, createdBy string) (Credential, error) {
	enc, err := s.cipher.Encrypt([]byte(token))
	if err != nil {
		return Credential{}, err
	}
	c, err := s.queries.CreateGitCredential(ctx, db.CreateGitCredentialParams{
		Name:      name,
		Username:  username,
		TokenEnc:  enc,
		CreatedBy: createdBy,
	})
	if err != nil {
		return Credential{}, err
	}
	return toCredential(c), nil
}

func (s *Store) Update(ctx context.Context, id int64, name, username, token string) (Credential, error) {
	var enc []byte
	if token == "" {
		existing, err := s.queries.GetGitCredential(ctx, id)
		if err != nil {
			return Credential{}, err
		}
		enc = existing.TokenEnc
	} else {
		var err error
		enc, err = s.cipher.Encrypt([]byte(token))
		if err != nil {
			return Credential{}, err
		}
	}

	c, err := s.queries.UpdateGitCredential(ctx, db.UpdateGitCredentialParams{
		ID:       id,
		Name:     name,
		Username: username,
		TokenEnc: enc,
	})
	if err != nil {
		return Credential{}, err
	}
	return toCredential(c), nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	return s.queries.DeleteGitCredential(ctx, id)
}

func (s *Store) Credentials(ctx context.Context, id int64) (username, token string, err error) {
	c, err := s.queries.GetGitCredential(ctx, id)
	if err != nil {
		return "", "", err
	}
	tok, err := s.cipher.Decrypt(c.TokenEnc)
	if err != nil {
		return "", "", err
	}
	return c.Username, string(tok), nil
}

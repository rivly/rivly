package registry

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/distribution/reference"
	moby "github.com/moby/moby/api/types/registry"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/secret"
)

type Registry struct {
	ID        int64
	Name      string
	Server    string
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

func toRegistry(r db.Registry) Registry {
	return Registry{
		ID:        r.ID,
		Name:      r.Name,
		Server:    r.Server,
		Username:  r.Username,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (s *Store) List(ctx context.Context) ([]Registry, error) {
	rows, err := s.queries.ListRegistries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Registry, 0, len(rows))
	for _, r := range rows {
		out = append(out, toRegistry(r))
	}
	return out, nil
}

func (s *Store) Create(ctx context.Context, name, server, username, password, createdBy string) (Registry, error) {
	enc, err := s.cipher.Encrypt([]byte(password))
	if err != nil {
		return Registry{}, err
	}
	r, err := s.queries.CreateRegistry(ctx, db.CreateRegistryParams{
		Name:        name,
		Server:      server,
		Username:    username,
		PasswordEnc: enc,
		CreatedBy:   createdBy,
	})
	if err != nil {
		return Registry{}, err
	}
	return toRegistry(r), nil
}

func (s *Store) Update(ctx context.Context, id int64, name, server, username, password string) (Registry, error) {
	var enc []byte
	if password == "" {
		existing, err := s.queries.GetRegistry(ctx, id)
		if err != nil {
			return Registry{}, err
		}
		enc = existing.PasswordEnc
	} else {
		var err error
		enc, err = s.cipher.Encrypt([]byte(password))
		if err != nil {
			return Registry{}, err
		}
	}

	r, err := s.queries.UpdateRegistry(ctx, db.UpdateRegistryParams{
		ID:          id,
		Name:        name,
		Server:      server,
		Username:    username,
		PasswordEnc: enc,
	})
	if err != nil {
		return Registry{}, err
	}
	return toRegistry(r), nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	return s.queries.DeleteRegistry(ctx, id)
}

func (s *Store) Credentials(ctx context.Context, id int64) (server, username, password string, err error) {
	r, err := s.queries.GetRegistry(ctx, id)
	if err != nil {
		return "", "", "", err
	}
	pw, err := s.cipher.Decrypt(r.PasswordEnc)
	if err != nil {
		return "", "", "", err
	}
	return r.Server, r.Username, string(pw), nil
}

func (s *Store) AuthFor(ctx context.Context, imageRef string) string {
	domain := Domain(imageRef)
	if domain == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	r, err := s.queries.GetRegistryByServer(ctx, domain)
	if err != nil {
		return ""
	}
	pw, err := s.cipher.Decrypt(r.PasswordEnc)
	if err != nil {
		return ""
	}
	auth, err := EncodeAuth(r.Server, r.Username, string(pw))
	if err != nil {
		return ""
	}
	return auth
}

func Domain(imageRef string) string {
	named, err := reference.ParseNormalizedNamed(imageRef)
	if err != nil {
		return ""
	}
	return reference.Domain(named)
}

func EncodeAuth(server, username, password string) (string, error) {
	cfg := moby.AuthConfig{Username: username, Password: password, ServerAddress: server}
	buf, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

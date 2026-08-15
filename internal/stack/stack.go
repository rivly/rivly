package stack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	"github.com/rivly/rivly/internal/compose"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
	"github.com/rivly/rivly/internal/gitrepo"
)

const (
	SourceContent = "content"
	SourceGit     = "git"
)

const (
	MinPollInterval = 15
	GitOpsAuthor    = "GitOps"
	maxComposeError = 4000
)

type DockerClient interface {
	Stacks(ctx context.Context) ([]docker.Stack, error)
	StackAction(ctx context.Context, project, action string) error
}

type Docker func(envID int64, host string) (DockerClient, error)

type Compose interface {
	Deploy(ctx context.Context, st compose.Stack) (string, error)
	Remove(ctx context.Context, st compose.Stack) (string, error)
	Discard(ctx context.Context, st compose.Stack)
	RepoDir(envID int64, project string) string
}

type Credentials interface {
	Credentials(ctx context.Context, id int64) (username, token string, err error)
}

type Publisher func(ctx context.Context, env db.Environment)

type Spawner func(fn func())

type Service struct {
	logger   *slog.Logger
	queries  *db.Queries
	docker   Docker
	compose  Compose
	gitcreds Credentials
	publish  Publisher
	spawn    Spawner

	mu       sync.Mutex
	inflight map[int64]bool
}

func New(
	logger *slog.Logger,
	queries *db.Queries,
	dockerClient Docker,
	composeRunner Compose,
	gitcreds Credentials,
	publish Publisher,
	spawn Spawner,
) *Service {
	return &Service{
		logger:   logger,
		queries:  queries,
		docker:   dockerClient,
		compose:  composeRunner,
		gitcreds: gitcreds,
		publish:  publish,
		spawn:    spawn,
		inflight: make(map[int64]bool),
	}
}

func (s *Service) Acquire(id int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inflight[id] {
		return false
	}
	s.inflight[id] = true
	return true
}

func (s *Service) Release(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inflight, id)
}

type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func EnvFileContent(vars []EnvVar) string {
	var b strings.Builder
	for _, v := range vars {
		key := strings.TrimSpace(v.Key)
		if key == "" {
			continue
		}
		fmt.Fprintf(&b, "%s=%s\n", key, quoteEnvValue(v.Value))
	}
	return b.String()
}

func quoteEnvValue(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func validateEnv(vars []EnvVar) error {
	for _, v := range vars {
		key := strings.TrimSpace(v.Key)
		if key == "" {
			continue
		}
		if !envKeyPattern.MatchString(key) {
			return invalid(fmt.Sprintf("%q is not a valid environment variable name", key))
		}
	}
	return nil
}

func ParseEnvVars(stored string) []EnvVar {
	vars := []EnvVar{}
	if stored != "" {
		_ = json.Unmarshal([]byte(stored), &vars)
	}
	return vars
}

func GitErrorMessage(err error) string {
	switch {
	case errors.Is(err, gitrepo.ErrCredentialsInURL):
		return gitrepo.ErrCredentialsInURL.Error()
	case errors.Is(err, gitrepo.ErrAuth):
		return "could not authenticate with the repository, check the credential"
	case errors.Is(err, gitrepo.ErrNotFound):
		return "repository not found"
	case errors.Is(err, gitrepo.ErrRef):
		return "branch or tag not found"
	}
	return "could not clone the repository"
}

func ComposeErrorMessage(out string) string {
	if out == "" {
		return "deployment failed"
	}
	if len(out) > maxComposeError {
		out = out[len(out)-maxComposeError:]
	}
	return out
}

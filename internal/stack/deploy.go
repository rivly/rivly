package stack

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/gitrepo"
)

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

type GitSource struct {
	URL          string
	Ref          string
	Path         string
	CredentialID int64
	AutoUpdate   bool
	PollInterval int64
}

type DeployInput struct {
	Name    string
	Source  string
	Content string
	Env     []EnvVar
	Git     *GitSource
	Author  string
}

func (s *Service) Deploy(ctx context.Context, env db.Environment, in DeployInput) error {
	name := strings.TrimSpace(in.Name)
	if !namePattern.MatchString(name) {
		return invalid("name must be lowercase letters, digits, - or _")
	}

	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = SourceContent
	}
	if source != SourceContent && source != SourceGit {
		return invalid("invalid stack source")
	}

	existing, getErr := s.queries.GetStack(ctx, db.GetStackParams{EnvID: env.ID, Name: name})
	if getErr != nil && !errors.Is(getErr, sql.ErrNoRows) {
		return fmt.Errorf("load stack %q: %w", name, getErr)
	}
	isNew := errors.Is(getErr, sql.ErrNoRows)

	envJSON, err := json.Marshal(in.Env)
	if err != nil {
		return invalid("invalid environment variables")
	}
	envContent := EnvFileContent(in.Env)

	params := db.UpsertStackParams{
		EnvID:     env.ID,
		Name:      name,
		Env:       string(envJSON),
		CreatedBy: in.Author,
		UpdatedBy: in.Author,
		Source:    source,
	}

	if source == SourceGit {
		if err := s.deployGit(ctx, env, name, existing.ID, &params, in.Git, envContent, isNew); err != nil {
			return err
		}
	} else {
		if strings.TrimSpace(in.Content) == "" {
			return invalid("compose file is empty")
		}
		out, derr := s.compose.Deploy(ctx, env.Url, env.ID, name, in.Content, envContent)
		if derr != nil {
			s.logger.Warn("stack deploy failed", "stack", name, "err", derr)
			if isNew {
				s.compose.Discard(ctx, env.Url, env.ID, name)
			}
			return rejected(ComposeErrorMessage(out))
		}
		params.Content = in.Content
	}

	if _, err := s.queries.UpsertStack(ctx, params); err != nil {
		return fmt.Errorf("save stack %q: %w", name, err)
	}
	return nil
}

func (s *Service) deployGit(
	ctx context.Context,
	env db.Environment,
	name string,
	stackID int64,
	params *db.UpsertStackParams,
	src *GitSource,
	envContent string,
	isNew bool,
) error {
	if src == nil {
		return invalid("git settings are required")
	}

	if stackID != 0 {
		if !s.Acquire(stackID) {
			return conflict("an update is already running for this stack")
		}
		defer s.Release(stackID)
	}

	repoURL, err := gitrepo.NormalizeURL(src.URL)
	if err != nil {
		return invalid(err.Error())
	}
	path, err := gitrepo.ComposePath(src.Path)
	if err != nil {
		return invalid(err.Error())
	}

	opts := gitrepo.Options{URL: repoURL, Ref: strings.TrimSpace(src.Ref)}
	if src.CredentialID != 0 {
		username, token, cerr := s.gitcreds.Credentials(ctx, src.CredentialID)
		if cerr != nil {
			return invalid("git credential not found")
		}
		opts.Username, opts.Token = username, token
	}

	remoteHash, err := gitrepo.RemoteHash(ctx, opts)
	if err != nil {
		s.logger.Warn("stack remote check failed", "stack", name, "url", repoURL, "err", err)
		return rejected(GitErrorMessage(err))
	}

	repoDir := s.compose.RepoDir(env.ID, name)
	commit, err := gitrepo.Clone(ctx, repoDir, opts)
	if err != nil {
		s.logger.Warn("stack clone failed", "stack", name, "url", repoURL, "err", err)
		return rejected(GitErrorMessage(err))
	}

	content, err := os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(path)))
	if err != nil {
		s.logger.Warn("compose file missing in repository", "stack", name, "path", path, "err", err)
		return rejected("compose file not found in the repository")
	}

	out, derr := s.compose.DeployRepo(ctx, env.Url, env.ID, name, path, envContent)
	if derr != nil {
		s.logger.Warn("git stack deploy failed", "stack", name, "err", derr)
		if isNew {
			s.compose.DiscardRepo(ctx, env.Url, env.ID, name, path)
		}
		return rejected(ComposeErrorMessage(out))
	}

	interval := src.PollInterval
	if interval < MinPollInterval {
		interval = MinPollInterval
	}

	params.Content = string(content)
	params.GitUrl = repoURL
	params.GitRef = strings.TrimSpace(src.Ref)
	params.GitPath = path
	params.GitCredentialID = src.CredentialID
	params.GitCommit = commit
	params.GitRemoteHash = remoteHash
	params.GitPollInterval = interval
	if src.AutoUpdate {
		params.GitAutoUpdate = 1
	}
	return nil
}

package server

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/gitrepo"
)

const (
	gitPollTick        = 5 * time.Second
	gitMinPollInterval = 15
	gitOpsAuthor       = "GitOps"
)

func (s *Server) RunGitPoller(ctx context.Context) {
	ticker := time.NewTicker(gitPollTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollGitStacks(ctx)
		}
	}
}

func (s *Server) pollGitStacks(ctx context.Context) {
	stacks, err := s.queries.ListAutoUpdateStacks(ctx)
	if err != nil {
		s.logger.Error("git poller: could not list stacks", "err", err)
		return
	}

	now := time.Now().Unix()
	for _, stack := range stacks {
		interval := stack.GitPollInterval
		if interval < gitMinPollInterval {
			interval = gitMinPollInterval
		}
		if now-stack.GitLastCheckedAt < interval {
			continue
		}
		if !s.acquireGitStack(stack.ID) {
			continue
		}
		go func(stack db.Stack) {
			defer s.releaseGitStack(stack.ID)
			s.checkGitStack(ctx, stack)
		}(stack)
	}
}

func (s *Server) acquireGitStack(id int64) bool {
	s.gitMu.Lock()
	defer s.gitMu.Unlock()
	if s.gitInflight[id] {
		return false
	}
	s.gitInflight[id] = true
	return true
}

func (s *Server) releaseGitStack(id int64) {
	s.gitMu.Lock()
	defer s.gitMu.Unlock()
	delete(s.gitInflight, id)
}

func (s *Server) checkGitStack(ctx context.Context, stack db.Stack) {
	opts := gitrepo.Options{URL: stack.GitUrl, Ref: stack.GitRef}
	if stack.GitCredentialID != 0 {
		username, token, err := s.gitcreds.Credentials(ctx, stack.GitCredentialID)
		if err != nil {
			s.markChecked(ctx, stack, "git credential not found", stack.GitRemoteHash)
			return
		}
		opts.Username, opts.Token = username, token
	}

	remoteHash, err := gitrepo.RemoteHash(ctx, opts)
	if err != nil {
		s.logger.Warn("git poller: check failed", "stack", stack.Name, "err", err)
		s.markChecked(ctx, stack, gitError(err), stack.GitRemoteHash)
		return
	}
	if remoteHash == stack.GitRemoteHash {
		s.markChecked(ctx, stack, "", remoteHash)
		return
	}

	env, err := s.queries.GetEnvironment(ctx, stack.EnvID)
	if err != nil {
		s.markChecked(ctx, stack, "environment not found", stack.GitRemoteHash)
		return
	}

	running, err := s.stackIsRunning(ctx, env, stack.Name)
	if err != nil {
		s.logger.Warn("git poller: could not read stack state", "stack", stack.Name, "err", err)
		s.markChecked(ctx, stack, "environment is unreachable", stack.GitRemoteHash)
		return
	}
	if !running {
		s.logger.Info("git poller: stack is stopped, skipping redeploy", "stack", stack.Name)
		s.markChecked(ctx, stack, "", remoteHash)
		return
	}

	s.redeployGitStack(ctx, env, stack, opts, remoteHash)
}

func (s *Server) stackIsRunning(ctx context.Context, env db.Environment, name string) (bool, error) {
	discovered, err := s.docker.Stacks(ctx, env.ID, env.Url)
	if err != nil {
		return false, err
	}
	for _, d := range discovered {
		if d.Name == name {
			return d.Running > 0, nil
		}
	}
	return false, nil
}

func (s *Server) redeployGitStack(ctx context.Context, env db.Environment, stack db.Stack, opts gitrepo.Options, remoteHash string) {
	repoDir := s.compose.RepoDir(env.ID, stack.Name)
	commit, err := gitrepo.Clone(ctx, repoDir, opts)
	if err != nil {
		s.logger.Warn("git poller: clone failed", "stack", stack.Name, "err", err)
		s.markChecked(ctx, stack, gitError(err), stack.GitRemoteHash)
		return
	}

	content, err := os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(stack.GitPath)))
	if err != nil {
		s.markChecked(ctx, stack, "compose file not found in the repository", stack.GitRemoteHash)
		return
	}

	out, err := s.compose.DeployRepo(ctx, env.Url, env.ID, stack.Name, stack.GitPath, envFileContent(parseEnvVars(stack.Env)))
	if err != nil {
		s.logger.Warn("git poller: deploy failed", "stack", stack.Name, "err", err, "out", out)
		s.markChecked(ctx, stack, composeError(out), stack.GitRemoteHash)
		return
	}

	if err := s.queries.ApplyStackGitUpdate(ctx, db.ApplyStackGitUpdateParams{
		ID:            stack.ID,
		Content:       string(content),
		GitCommit:     commit,
		GitRemoteHash: remoteHash,
		UpdatedBy:     gitOpsAuthor,
	}); err != nil {
		s.logger.Error("git poller: could not save stack", "stack", stack.Name, "err", err)
		return
	}

	s.logger.Info("git poller: stack updated", "stack", stack.Name, "commit", commit)
	s.publishEnvironment(ctx, env)
}

func (s *Server) markChecked(ctx context.Context, stack db.Stack, message, remoteHash string) {
	if err := s.queries.MarkStackChecked(ctx, db.MarkStackCheckedParams{
		ID:            stack.ID,
		GitLastError:  message,
		GitRemoteHash: remoteHash,
	}); err != nil {
		s.logger.Error("git poller: could not mark checked", "stack", stack.Name, "err", err)
	}
}

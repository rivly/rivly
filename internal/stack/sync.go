package stack

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/rivly/rivly/internal/compose"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/gitrepo"
)

const syncTick = 5 * time.Second

func (s *Service) RunSync(ctx context.Context) {
	ticker := time.NewTicker(syncTick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Sync(ctx)
		}
	}
}

func (s *Service) Sync(ctx context.Context) {
	stacks, err := s.queries.ListAutoUpdateStacks(ctx)
	if err != nil {
		s.logger.Error("stack sync: could not list stacks", "err", err)
		return
	}

	now := time.Now().Unix()
	for _, st := range stacks {
		interval := st.GitPollInterval
		if interval < MinPollInterval {
			interval = MinPollInterval
		}
		if now-st.GitLastCheckedAt < interval {
			continue
		}
		if !s.Acquire(st.ID) {
			continue
		}
		s.spawn(func() {
			defer s.Release(st.ID)
			s.checkStack(ctx, st)
		})
	}
}

func (s *Service) checkStack(ctx context.Context, st db.Stack) {
	opts := gitrepo.Options{URL: st.GitUrl, Ref: st.GitRef}
	if st.GitCredentialID != 0 {
		username, token, err := s.gitcreds.Credentials(ctx, st.GitCredentialID)
		if err != nil {
			s.markChecked(ctx, st, "git credential not found", st.GitRemoteHash)
			return
		}
		opts.Username, opts.Token = username, token
	}

	remoteHash, err := gitrepo.RemoteHash(ctx, opts)
	if err != nil {
		s.logger.Warn("stack sync: check failed", "stack", st.Name, "err", err)
		s.markChecked(ctx, st, GitErrorMessage(err), st.GitRemoteHash)
		return
	}
	if remoteHash == st.GitRemoteHash {
		s.markChecked(ctx, st, "", remoteHash)
		return
	}

	env, err := s.queries.GetEnvironment(ctx, st.EnvID)
	if err != nil {
		s.markChecked(ctx, st, "environment not found", st.GitRemoteHash)
		return
	}

	running, err := s.isRunning(ctx, env, st.Name)
	if err != nil {
		s.logger.Warn("stack sync: could not read stack state", "stack", st.Name, "err", err)
		s.markChecked(ctx, st, "environment is unreachable", st.GitRemoteHash)
		return
	}
	if !running {
		s.logger.Info("stack sync: stack is stopped, skipping redeploy", "stack", st.Name)
		s.markChecked(ctx, st, "", remoteHash)
		return
	}

	s.redeploy(ctx, env, st, opts, remoteHash)
}

func (s *Service) isRunning(ctx context.Context, env db.Environment, name string) (bool, error) {
	client, err := s.docker(env.ID, env.Url)
	if err != nil {
		return false, err
	}
	discovered, err := client.Stacks(ctx)
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

func (s *Service) redeploy(ctx context.Context, env db.Environment, st db.Stack, opts gitrepo.Options, remoteHash string) {
	repoDir := s.compose.RepoDir(env.ID, st.Name)
	commit, err := gitrepo.Clone(ctx, repoDir, opts)
	if err != nil {
		s.logger.Warn("stack sync: clone failed", "stack", st.Name, "err", err)
		s.markChecked(ctx, st, GitErrorMessage(err), st.GitRemoteHash)
		return
	}

	content, err := os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(st.GitPath)))
	if err != nil {
		s.markChecked(ctx, st, "compose file not found in the repository", st.GitRemoteHash)
		return
	}

	out, err := s.compose.Deploy(ctx, compose.Stack{Source: compose.Repository, DockerHost: env.Url, EnvID: env.ID, Project: st.Name, File: st.GitPath, Env: EnvFileContent(ParseEnvVars(st.Env))})
	if err != nil {
		s.logger.Warn("stack sync: deploy failed", "stack", st.Name, "err", err, "out", out)
		s.markChecked(ctx, st, ComposeErrorMessage(out), st.GitRemoteHash)
		return
	}

	if err := s.queries.ApplyStackGitUpdate(ctx, db.ApplyStackGitUpdateParams{
		ID:            st.ID,
		Content:       string(content),
		GitCommit:     commit,
		GitRemoteHash: remoteHash,
		UpdatedBy:     GitOpsAuthor,
	}); err != nil {
		s.logger.Error("stack sync: could not save stack", "stack", st.Name, "err", err)
		return
	}

	s.logger.Info("stack sync: stack updated", "stack", st.Name, "commit", commit)
	s.publish(ctx, env)
}

func (s *Service) markChecked(ctx context.Context, st db.Stack, message, remoteHash string) {
	if err := s.queries.MarkStackChecked(ctx, db.MarkStackCheckedParams{
		ID:            st.ID,
		GitLastError:  message,
		GitRemoteHash: remoteHash,
	}); err != nil {
		s.logger.Error("stack sync: could not mark checked", "stack", st.Name, "err", err)
	}
}

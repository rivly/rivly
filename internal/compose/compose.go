package compose

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	composeTimeout = 5 * time.Minute
	composeFile    = "docker-compose.yml"
	envFile        = ".env"
	repoSubdir     = "repo"
)

type Runner struct {
	bin     string
	dataDir string
}

func NewRunner(bin, dataDir string) *Runner {
	return &Runner{bin: bin, dataDir: dataDir}
}

func (r *Runner) Deploy(ctx context.Context, dockerHost string, envID int64, project, content, env string) (string, error) {
	dir, err := r.materialize(envID, project, content, env)
	if err != nil {
		return "", err
	}
	if out, perr := r.pull(ctx, dockerHost, dir, project, composeFile, ""); perr != nil {
		return out, perr
	}
	return r.run(ctx, dockerHost, dir, project, composeFile, "", "up", "-d", "--remove-orphans")
}

func (r *Runner) Remove(ctx context.Context, dockerHost string, envID int64, project, content, env string) (string, error) {
	dir, err := r.materialize(envID, project, content, env)
	if err != nil {
		return "", err
	}
	out, rerr := r.run(ctx, dockerHost, dir, project, composeFile, "", "down", "--remove-orphans")
	if rerr == nil {
		_ = os.RemoveAll(dir)
	}
	return out, rerr
}

func (r *Runner) Discard(ctx context.Context, dockerHost string, envID int64, project string) {
	dir := r.projectDir(envID, project)
	_, _ = r.run(ctx, dockerHost, dir, project, composeFile, "", "down", "--remove-orphans")
	_ = os.RemoveAll(dir)
}

func (r *Runner) RepoDir(envID int64, project string) string {
	return filepath.Join(r.projectDir(envID, project), repoSubdir)
}

func (r *Runner) DeployRepo(ctx context.Context, dockerHost string, envID int64, project, file, env string) (string, error) {
	envPath, err := r.writeRepoEnv(envID, project, env)
	if err != nil {
		return "", err
	}
	dir := r.RepoDir(envID, project)
	if out, perr := r.pull(ctx, dockerHost, dir, project, file, envPath); perr != nil {
		return out, perr
	}
	return r.run(ctx, dockerHost, dir, project, file, envPath, "up", "-d", "--remove-orphans")
}

func (r *Runner) RemoveRepo(ctx context.Context, dockerHost string, envID int64, project, file, env string) (string, error) {
	envPath, err := r.writeRepoEnv(envID, project, env)
	if err != nil {
		return "", err
	}
	out, rerr := r.run(ctx, dockerHost, r.RepoDir(envID, project), project, file, envPath, "down", "--remove-orphans")
	if rerr == nil {
		_ = os.RemoveAll(r.projectDir(envID, project))
	}
	return out, rerr
}

func (r *Runner) DiscardRepo(ctx context.Context, dockerHost string, envID int64, project, file string) {
	_, _ = r.run(ctx, dockerHost, r.RepoDir(envID, project), project, file, "", "down", "--remove-orphans")
	_ = os.RemoveAll(r.projectDir(envID, project))
}

func (r *Runner) projectDir(envID int64, project string) string {
	return filepath.Join(r.dataDir, "stacks", strconv.FormatInt(envID, 10), project)
}

func (r *Runner) materialize(envID int64, project, content, env string) (string, error) {
	dir := r.projectDir(envID, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, composeFile), []byte(content), 0o600); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, envFile), []byte(env), 0o600); err != nil {
		return "", err
	}
	return dir, nil
}

func (r *Runner) writeRepoEnv(envID int64, project, env string) (string, error) {
	if strings.TrimSpace(env) == "" {
		return "", nil
	}
	dir := r.projectDir(envID, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(dir, envFile))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(env), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (r *Runner) pull(ctx context.Context, dockerHost, dir, project, file, envPath string) (string, error) {
	return r.run(ctx, dockerHost, dir, project, file, envPath, "pull", "--ignore-buildable")
}

func (r *Runner) run(ctx context.Context, dockerHost, dir, project, file, envPath string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, composeTimeout)
	defer cancel()

	full := []string{"-p", project, "-f", file}
	if envPath != "" {
		full = append(full, "--env-file", envPath)
	}
	full = append(full, args...)

	cmd := exec.CommandContext(ctx, r.bin, full...)
	cmd.Dir = dir
	cmd.Env = withDockerHost(os.Environ(), dockerHost)

	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func withDockerHost(env []string, dockerHost string) []string {
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if !strings.HasPrefix(e, "DOCKER_HOST=") {
			out = append(out, e)
		}
	}
	return append(out, "DOCKER_HOST="+dockerHost)
}

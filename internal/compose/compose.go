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
)

type Runner struct {
	bin     string
	dataDir string
}

func NewRunner(bin, dataDir string) *Runner {
	return &Runner{bin: bin, dataDir: dataDir}
}

func (r *Runner) Deploy(ctx context.Context, dockerHost string, envID int64, project, content string) (string, error) {
	dir, err := r.materialize(envID, project, content)
	if err != nil {
		return "", err
	}
	return r.run(ctx, dockerHost, dir, project, "up", "-d", "--remove-orphans")
}

func (r *Runner) Remove(ctx context.Context, dockerHost string, envID int64, project, content string) (string, error) {
	dir, err := r.materialize(envID, project, content)
	if err != nil {
		return "", err
	}
	out, rerr := r.run(ctx, dockerHost, dir, project, "down", "--remove-orphans")
	if rerr == nil {
		_ = os.RemoveAll(dir)
	}
	return out, rerr
}

func (r *Runner) Discard(ctx context.Context, dockerHost string, envID int64, project string) {
	dir := r.projectDir(envID, project)
	_, _ = r.run(ctx, dockerHost, dir, project, "down", "--remove-orphans")
	_ = os.RemoveAll(dir)
}

func (r *Runner) projectDir(envID int64, project string) string {
	return filepath.Join(r.dataDir, "stacks", strconv.FormatInt(envID, 10), project)
}

func (r *Runner) materialize(envID int64, project, content string) (string, error) {
	dir := r.projectDir(envID, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, composeFile), []byte(content), 0o600); err != nil {
		return "", err
	}
	return dir, nil
}

func (r *Runner) run(ctx context.Context, dockerHost, dir, project string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, composeTimeout)
	defer cancel()

	full := append([]string{"-p", project, "-f", composeFile}, args...)
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

package compose

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	composeTimeout = 5 * time.Minute
	probeTimeout   = 5 * time.Second
	dataDirMarker  = "<data>"
	composeFile    = "docker-compose.yml"
	envFile        = ".env"
	repoSubdir     = "repo"
)

var ErrNoCompose = errors.New("no docker compose executable found, set RIVLY_COMPOSE_BIN")

var candidates = [][]string{
	{"docker", "compose"},
	{"docker-compose"},
}

type Runner struct {
	command []string
	dataDir string
}

func NewRunner(ctx context.Context, bin, dataDir string) *Runner {
	return &Runner{command: resolveCommand(ctx, bin), dataDir: dataDir}
}

func (r *Runner) Command() string {
	return strings.Join(r.command, " ")
}

func parseCommand(bin string) []string {
	return strings.Fields(bin)
}

func resolveCommand(ctx context.Context, bin string) []string {
	if override := parseCommand(bin); len(override) > 0 {
		if _, err := exec.LookPath(override[0]); err != nil {
			return nil
		}
		return override
	}
	for _, candidate := range candidates {
		if available(ctx, candidate) {
			return candidate
		}
	}
	return nil
}

func available(ctx context.Context, command []string) bool {
	if _, err := exec.LookPath(command[0]); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	args := make([]string, 0, len(command))
	args = append(args, command[1:]...)
	args = append(args, "version")
	return exec.CommandContext(ctx, command[0], args...).Run() == nil
}

type Source int

const (
	Inline Source = iota
	Repository
)

type Stack struct {
	Source     Source
	DockerHost string
	EnvID      int64
	Project    string
	Content    string
	File       string
	Env        string
}

func (r *Runner) Deploy(ctx context.Context, st Stack) (string, error) {
	dir, file, envPath, err := r.prepare(st)
	if err != nil {
		return "", err
	}
	if out, perr := r.pull(ctx, st.DockerHost, dir, st.Project, file, envPath); perr != nil {
		return out, perr
	}
	return r.run(ctx, st.DockerHost, dir, st.Project, file, envPath, "up", "-d", "--remove-orphans")
}

func (r *Runner) Remove(ctx context.Context, st Stack) (string, error) {
	dir, file, envPath, err := r.prepare(st)
	if err != nil {
		return "", err
	}
	out, rerr := r.run(ctx, st.DockerHost, dir, st.Project, file, envPath, "down", "--remove-orphans")
	if rerr == nil {
		_ = os.RemoveAll(r.projectDir(st.EnvID, st.Project))
	}
	return out, rerr
}

func (r *Runner) Discard(ctx context.Context, st Stack) {
	dir, file := r.location(st)
	_, _ = r.run(ctx, st.DockerHost, dir, st.Project, file, "", "down", "--remove-orphans")
	_ = os.RemoveAll(r.projectDir(st.EnvID, st.Project))
}

func (r *Runner) RepoDir(envID int64, project string) string {
	return filepath.Join(r.projectDir(envID, project), repoSubdir)
}

func (r *Runner) location(st Stack) (dir, file string) {
	if st.Source == Repository {
		return r.RepoDir(st.EnvID, st.Project), st.File
	}
	return r.projectDir(st.EnvID, st.Project), composeFile
}

func (r *Runner) prepare(st Stack) (dir, file, envPath string, err error) {
	dir, file = r.location(st)
	if st.Source == Repository {
		envPath, err = r.writeRepoEnv(st.EnvID, st.Project, st.Env)
		return dir, file, envPath, err
	}
	if err := r.materialize(st.EnvID, st.Project, st.Content, st.Env); err != nil {
		return "", "", "", err
	}
	return dir, file, "", nil
}

func (r *Runner) projectDir(envID int64, project string) string {
	return filepath.Join(r.dataDir, "stacks", strconv.FormatInt(envID, 10), project)
}

func (r *Runner) materialize(envID int64, project, content, env string) error {
	dir := r.projectDir(envID, project)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, composeFile), []byte(content), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, envFile), []byte(env), 0o600)
}

func (r *Runner) writeRepoEnv(envID int64, project, env string) (string, error) {
	if strings.TrimSpace(env) == "" {
		return "", nil
	}
	dir := r.projectDir(envID, project)
	if err := os.MkdirAll(dir, 0o700); err != nil {
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
	if len(r.command) == 0 {
		return "", ErrNoCompose
	}

	ctx, cancel := context.WithTimeout(ctx, composeTimeout)
	defer cancel()

	full := make([]string, 0, len(r.command)+len(args)+6)
	full = append(full, r.command[1:]...)
	full = append(full, "-p", project, "-f", file)
	if envPath != "" {
		full = append(full, "--env-file", envPath)
	}
	full = append(full, args...)

	cmd := exec.CommandContext(ctx, r.command[0], full...)
	cmd.Dir = dir
	cmd.Env = withDockerHost(os.Environ(), dockerHost)

	out, err := cmd.CombinedOutput()
	return r.redact(strings.TrimSpace(string(out))), err
}

func (r *Runner) redact(out string) string {
	absolute, err := filepath.Abs(r.dataDir)
	if err != nil || absolute == string(filepath.Separator) {
		return out
	}
	return strings.ReplaceAll(out, absolute, dataDirMarker)
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

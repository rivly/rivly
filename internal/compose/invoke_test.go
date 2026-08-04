package compose

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func recordingCompose(t *testing.T) (*Runner, func() []string) {
	t.Helper()

	dir := t.TempDir()
	log := filepath.Join(dir, "calls.log")
	script := filepath.Join(dir, "docker-compose")

	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	runner := NewRunner(context.Background(), script, filepath.Join(dir, "data"))
	return runner, func() []string {
		raw, err := os.ReadFile(log)
		if err != nil {
			return nil
		}
		return strings.Split(strings.TrimSpace(string(raw)), "\n")
	}
}

func TestDeployPullsThenBringsTheStackUp(t *testing.T) {
	runner, recorded := recordingCompose(t)

	out, err := runner.Deploy(context.Background(), "unix:///run/docker.sock", 1, "app", "services: {}\n", "PORT=\"8080\"\n")
	if err != nil {
		t.Fatalf("Deploy: %v (%s)", err, out)
	}

	calls := recorded()
	if len(calls) != 2 {
		t.Fatalf("want a pull then an up, got %q", calls)
	}
	if !strings.Contains(calls[0], "pull --ignore-buildable") {
		t.Errorf("first call must pull: %q", calls[0])
	}
	if !strings.Contains(calls[1], "up -d --remove-orphans") {
		t.Errorf("second call must bring the stack up: %q", calls[1])
	}
	for _, c := range calls {
		if !strings.Contains(c, "-p app") {
			t.Errorf("the project name pins the compose project: %q", c)
		}
		if !strings.Contains(c, "-f docker-compose.yml") {
			t.Errorf("the compose file must be explicit: %q", c)
		}
	}
}

func TestDeployMaterialisesTheComposeAndEnvFiles(t *testing.T) {
	runner, _ := recordingCompose(t)

	const compose = "services:\n  web:\n    image: nginx\n"
	const env = "PORT=\"8080\"\n"
	if _, err := runner.Deploy(context.Background(), "unix:///run/docker.sock", 7, "app", compose, env); err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	dir := runner.projectDir(7, "app")
	for name, want := range map[string]string{
		composeFile: compose,
		envFile:     env,
	} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if string(got) != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	info, err := os.Stat(filepath.Join(dir, envFile))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("the env file holds secrets, mode = %o, want 600", perm)
	}
}

func TestRemoveTearsDownAndForgetsTheProject(t *testing.T) {
	runner, recorded := recordingCompose(t)

	if _, err := runner.Deploy(context.Background(), "unix:///run/docker.sock", 1, "app", "services: {}\n", ""); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	dir := runner.projectDir(1, "app")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("the project directory must exist after a deploy: %v", err)
	}

	if _, err := runner.Remove(context.Background(), "unix:///run/docker.sock", 1, "app", "services: {}\n", ""); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if !strings.Contains(recorded()[len(recorded())-1], "down --remove-orphans") {
		t.Errorf("remove must tear the stack down: %q", recorded())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("a removed stack must leave nothing on disk, got %v", err)
	}
}

func TestDeployRepoPassesTheEnvFileExplicitly(t *testing.T) {
	runner, recorded := recordingCompose(t)

	repo := runner.RepoDir(1, "app")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if _, err := runner.DeployRepo(context.Background(), "unix:///run/docker.sock", 1, "app", "deploy/compose.yml", "TOKEN=\"x\"\n"); err != nil {
		t.Fatalf("DeployRepo: %v", err)
	}

	for _, c := range recorded() {
		if !strings.Contains(c, "-f deploy/compose.yml") {
			t.Errorf("the compose path inside the repository must be used: %q", c)
		}
		if !strings.Contains(c, "--env-file") {
			t.Errorf("a git stack keeps its env file outside the repository: %q", c)
		}
	}
}

func TestDeployRepoOmitsTheEnvFileWhenThereIsNone(t *testing.T) {
	runner, recorded := recordingCompose(t)

	repo := runner.RepoDir(1, "app")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if _, err := runner.DeployRepo(context.Background(), "unix:///run/docker.sock", 1, "app", "compose.yml", "   "); err != nil {
		t.Fatalf("DeployRepo: %v", err)
	}

	for _, c := range recorded() {
		if strings.Contains(c, "--env-file") {
			t.Errorf("an empty env must not produce an empty file argument: %q", c)
		}
	}
}

func TestRunnerPointsComposeAtTheRightDaemon(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "env.log")
	script := filepath.Join(dir, "docker-compose")

	body := "#!/bin/sh\nprintf '%s\\n' \"$DOCKER_HOST\" >> " + log + "\n"
	if err := os.WriteFile(script, []byte(body), 0o700); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("DOCKER_HOST", "unix:///wrong/inherited.sock")
	runner := NewRunner(context.Background(), script, filepath.Join(dir, "data"))

	if _, err := runner.Deploy(context.Background(), "tcp://prod:2376", 1, "app", "services: {}\n", ""); err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line != "tcp://prod:2376" {
			t.Fatalf("compose must target the environment, not the inherited host: %q", line)
		}
	}
}

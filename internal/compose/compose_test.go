package compose

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		bin  string
		want []string
	}{
		{"v2 plugin as a subcommand", "docker compose", []string{"docker", "compose"}},
		{"v1 standalone binary", "docker-compose", []string{"docker-compose"}},
		{"absolute path to the plugin", "/usr/libexec/docker/cli-plugins/docker-compose", []string{"/usr/libexec/docker/cli-plugins/docker-compose"}},
		{"surrounding whitespace", "   docker   compose   ", []string{"docker", "compose"}},
		{"a remote host wrapper", "docker --context prod compose", []string{"docker", "--context", "prod", "compose"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := parseCommand(tc.bin); !slices.Equal(got, tc.want) {
				t.Fatalf("parseCommand(%q) = %v, want %v", tc.bin, got, tc.want)
			}
		})
	}
}

func TestResolveCommandRejectsAnOverrideThatDoesNotExist(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "compose-that-is-not-there")
	if got := resolveCommand(missing); got != nil {
		t.Fatalf("a configured executable that does not exist must not be trusted, got %v", got)
	}
}

func TestRunnerReportsItsCommand(t *testing.T) {
	t.Parallel()

	if got := NewRunner("docker compose", t.TempDir()).Command(); got != "docker compose" {
		t.Fatalf("Command() = %q, want %q", got, "docker compose")
	}
}

func TestRunWithoutAComposeExecutableFailsCleanly(t *testing.T) {
	t.Parallel()

	runner := &Runner{dataDir: t.TempDir()}
	if got := runner.Command(); got != "" {
		t.Fatalf("Command() = %q, want empty", got)
	}

	out, err := runner.Deploy(context.Background(), "unix:///var/run/docker.sock", 1, "app", "services: {}\n", "")
	if !errors.Is(err, ErrNoCompose) {
		t.Fatalf("Deploy without compose: want ErrNoCompose, got %v", err)
	}
	if out != "" {
		t.Fatalf("Deploy without compose returned output %q", out)
	}
}

func TestWithDockerHostReplacesAnyInheritedValue(t *testing.T) {
	t.Parallel()

	got := withDockerHost([]string{"PATH=/bin", "DOCKER_HOST=tcp://stale:2375", "HOME=/root"}, "unix:///run/docker.sock")
	want := []string{"PATH=/bin", "HOME=/root", "DOCKER_HOST=unix:///run/docker.sock"}
	if !slices.Equal(got, want) {
		t.Fatalf("withDockerHost = %v, want %v", got, want)
	}
}

func TestRedactHidesTheDataDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runner := &Runner{dataDir: dir}

	out := runner.redact("services.web Additional property imagee is not allowed\n" +
		"file: " + filepath.Join(dir, "stacks", "1", "app", "docker-compose.yml"))

	if strings.Contains(out, dir) {
		t.Fatalf("the host path must not reach the client: %q", out)
	}
	if !strings.Contains(out, "<data>/stacks/1/app/docker-compose.yml") {
		t.Errorf("the useful part of the path must survive: %q", out)
	}
	if !strings.Contains(out, "Additional property imagee is not allowed") {
		t.Errorf("the compose error itself must survive: %q", out)
	}
}

func TestRedactWorksFromARelativeDataDirectory(t *testing.T) {
	t.Parallel()

	runner := &Runner{dataDir: "data"}
	absolute, err := filepath.Abs("data")
	if err != nil {
		t.Fatalf("Abs: %v", err)
	}

	out := runner.redact("cannot read " + filepath.Join(absolute, "stacks", "1", "app"))
	if strings.Contains(out, absolute) {
		t.Fatalf("a relative data dir must still be redacted by its absolute form: %q", out)
	}
}

func TestRedactLeavesOrdinaryOutputAlone(t *testing.T) {
	t.Parallel()

	runner := &Runner{dataDir: "data"}
	const message = "invalid data in service web: no such image"

	if got := runner.redact(message); got != message {
		t.Fatalf("the word data must not be substituted on its own: %q", got)
	}
}

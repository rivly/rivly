package compose

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestResolveCommandHonoursTheOverride(t *testing.T) {
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
			if got := resolveCommand(tc.bin); !slices.Equal(got, tc.want) {
				t.Fatalf("resolveCommand(%q) = %v, want %v", tc.bin, got, tc.want)
			}
		})
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

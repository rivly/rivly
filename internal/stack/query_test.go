package stack

import (
	"context"
	"errors"
	"testing"

	"github.com/rivly/rivly/internal/docker"
)

var errComposeFailed = errors.New("exit status 1")

func TestListMergesDiscoveredAndManagedStacks(t *testing.T) {
	h := newHarness(t, fakeDocker{stacks: []docker.Stack{
		{Name: "zulu", Type: "external", Services: 1, Running: 1, Total: 1, State: "running", WorkingDir: "/srv/zulu"},
		{Name: "alpha", Type: "external", Services: 2, Running: 1, Total: 2, State: "partial"},
	}}.factory(), fakeCompose{})

	h.seed(t, "alpha", fakeRemote(t, headA), headA, 15, 1)
	h.seed(t, "mike", fakeRemote(t, headA), headA, 15, 1)

	got, err := h.List(context.Background(), h.env(t))
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("want 3 stacks, got %d: %+v", len(got), got)
	}
	if got[0].Name != "alpha" || got[1].Name != "mike" || got[2].Name != "zulu" {
		t.Fatalf("stacks must be sorted by name, got %q %q %q", got[0].Name, got[1].Name, got[2].Name)
	}

	if got[0].Type != "rivly" || got[0].Running != 1 || got[0].WorkingDir != "" {
		t.Errorf("a discovered stack that Rivly manages keeps its live counts and becomes rivly: %+v", got[0])
	}
	if got[0].CreatedBy != "tester" {
		t.Errorf("a managed stack must carry its author, got %q", got[0].CreatedBy)
	}
	if got[1].Type != "rivly" || got[1].State != "stopped" {
		t.Errorf("a managed stack with no container must show as stopped: %+v", got[1])
	}
	if got[2].Type != "external" || got[2].WorkingDir != "/srv/zulu" {
		t.Errorf("an unmanaged stack stays external and keeps its working dir: %+v", got[2])
	}
}

func TestListReportsAnUnreachableDaemon(t *testing.T) {
	h := newHarness(t, fakeDocker{err: errComposeFailed}.factory(), fakeCompose{})

	_, err := h.List(context.Background(), h.env(t))
	if err == nil {
		t.Fatal("expected an error when the daemon is unreachable")
	}
	if got := kindOf(t, err); got != KindUnreachable {
		t.Fatalf("kind = %v, want KindUnreachable", got)
	}
}

func TestGetReturnsGitDetailsForAGitStack(t *testing.T) {
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{})
	h.seed(t, "app", "https://example.test/acme/app.git", headA, 300, 1)

	detail, err := h.Get(context.Background(), h.env(t), "app")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if detail.Git == nil {
		t.Fatal("a git stack must carry its git details")
	}
	if detail.Git.URL != "https://example.test/acme/app.git" || detail.Git.Ref != "main" {
		t.Errorf("git details = %+v", detail.Git)
	}
	if !detail.Git.AutoUpdate || detail.Git.PollInterval != 300 {
		t.Errorf("auto update settings were not returned: %+v", detail.Git)
	}
}

func TestGetReportsAMissingStack(t *testing.T) {
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{})

	_, err := h.Get(context.Background(), h.env(t), "nope")
	if got := kindOf(t, err); got != KindNotFound {
		t.Fatalf("kind = %v, want KindNotFound", got)
	}
}

func TestSignatureTracksManagedStacks(t *testing.T) {
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{})

	empty := h.Signature(context.Background(), 1)
	h.seed(t, "app", fakeRemote(t, headA), headA, 15, 1)
	withStack := h.Signature(context.Background(), 1)

	if empty == withStack {
		t.Fatal("the signature must change when a stack appears, otherwise no event is published")
	}
}

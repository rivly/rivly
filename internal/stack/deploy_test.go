package stack

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDeployRejectsInvalidInput(t *testing.T) {
	cases := map[string]struct {
		in DeployInput
	}{
		"empty name":            {DeployInput{Name: "", Content: "services: {}"}},
		"uppercase name":        {DeployInput{Name: "MyApp", Content: "services: {}"}},
		"leading dash":          {DeployInput{Name: "-app", Content: "services: {}"}},
		"path traversal name":   {DeployInput{Name: "../evil", Content: "services: {}"}},
		"name with a slash":     {DeployInput{Name: "a/b", Content: "services: {}"}},
		"name over 63 chars":    {DeployInput{Name: "a123456789012345678901234567890123456789012345678901234567890123", Content: "services: {}"}},
		"unknown source":        {DeployInput{Name: "app", Source: "svn", Content: "services: {}"}},
		"empty compose file":    {DeployInput{Name: "app", Content: "   "}},
		"git without settings":  {DeployInput{Name: "app", Source: SourceGit}},
		"credentials in theurl": {DeployInput{Name: "app", Source: SourceGit, Git: &GitSource{URL: "https://u:tok@host/r.git", Path: "c.yml"}}},
		"absolute compose path": {DeployInput{Name: "app", Source: SourceGit, Git: &GitSource{URL: "https://host/r.git", Path: "/etc/passwd"}}},
		"escaping compose path": {DeployInput{Name: "app", Source: SourceGit, Git: &GitSource{URL: "https://host/r.git", Path: "../../etc/passwd"}}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, fakeDocker{}.factory(), fakeCompose{})

			err := h.Deploy(context.Background(), h.env(t), tc.in)
			if err == nil {
				t.Fatal("expected the deploy to be rejected")
			}
			if got := kindOf(t, err); got != KindInvalid {
				t.Fatalf("kind = %v, want KindInvalid", got)
			}
		})
	}
}

func TestDeployStoresTheStack(t *testing.T) {
	deployed := make(chan string, 1)
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{deployed: deployed})

	in := DeployInput{
		Name:    "my-app",
		Content: "services:\n  web:\n    image: nginx\n",
		Env:     []EnvVar{{Key: "PORT", Value: "8080"}, {Key: "  ", Value: "ignored"}},
		Author:  "Mael",
	}
	if err := h.Deploy(context.Background(), h.env(t), in); err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	select {
	case project := <-deployed:
		if project != "my-app" {
			t.Errorf("compose was called for %q, want %q", project, "my-app")
		}
	default:
		t.Fatal("compose was never invoked")
	}

	record := h.reload(t, "my-app")
	if record.Source != SourceContent {
		t.Errorf("source = %q, want %q", record.Source, SourceContent)
	}
	if record.Content != in.Content {
		t.Errorf("content was not stored verbatim: %q", record.Content)
	}
	if record.CreatedBy != "Mael" || record.UpdatedBy != "Mael" {
		t.Errorf("author = %q/%q, want Mael", record.CreatedBy, record.UpdatedBy)
	}

	detail, err := h.Get(context.Background(), h.env(t), "my-app")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(detail.Env) != 2 {
		t.Fatalf("env vars = %+v, want the raw pair list", detail.Env)
	}
	if detail.Git != nil {
		t.Error("a content stack must not carry git details")
	}
}

func TestDeployReportsAComposeFailure(t *testing.T) {
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{
		out: "yaml: line 3: did not find expected key",
		err: errComposeFailed,
	})

	err := h.Deploy(context.Background(), h.env(t), DeployInput{
		Name:    "broken",
		Content: "services: bad",
	})
	if err == nil {
		t.Fatal("expected the deploy to fail")
	}
	if got := kindOf(t, err); got != KindRejected {
		t.Fatalf("kind = %v, want KindRejected", got)
	}
	if err.Error() != "yaml: line 3: did not find expected key" {
		t.Errorf("the compose output must reach the caller, got %q", err.Error())
	}

	if _, err := h.Get(context.Background(), h.env(t), "broken"); kindOf(t, err) != KindNotFound {
		t.Error("a failed deploy must not leave a stack behind")
	}
}

func TestDeployRefusesAStackBeingUpdated(t *testing.T) {
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{})

	existing := h.seed(t, "app", fakeRemote(t, headA), headA, 15, 1)
	if !h.Acquire(existing.ID) {
		t.Fatal("Acquire must succeed on a free stack")
	}
	defer h.Release(existing.ID)

	err := h.Deploy(context.Background(), h.env(t), DeployInput{
		Name:   "app",
		Source: SourceGit,
		Git:    &GitSource{URL: "https://host/r.git", Path: "docker-compose.yml"},
	})
	if err == nil {
		t.Fatal("expected a conflict")
	}
	if got := kindOf(t, err); got != KindConflict {
		t.Fatalf("kind = %v, want KindConflict", got)
	}
}

func TestDeploySwitchingFromGitToContentDropsTheCheckout(t *testing.T) {
	root := t.TempDir()
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{repoDir: root})
	h.seed(t, "app", fakeRemote(t, headA), headA, 15, 1)

	checkout := filepath.Join(root, "app")
	if err := os.MkdirAll(checkout, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(checkout, "docker-compose.yml"), []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := h.Deploy(context.Background(), h.env(t), DeployInput{
		Name:    "app",
		Source:  SourceContent,
		Content: "services:\n  web:\n    image: nginx\n",
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	if _, statErr := os.Stat(checkout); !os.IsNotExist(statErr) {
		t.Fatalf("the git checkout must be dropped when a stack stops being a git stack, got %v", statErr)
	}

	if got := h.reload(t, "app").Source; got != SourceContent {
		t.Errorf("source = %q, want %q", got, SourceContent)
	}
}

func TestDeployContentStackLeavesTheCheckoutAlone(t *testing.T) {
	root := t.TempDir()
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{repoDir: root})

	if err := h.Deploy(context.Background(), h.env(t), DeployInput{Name: "app", Content: "services: {}"}); err != nil {
		t.Fatalf("first Deploy: %v", err)
	}

	checkout := filepath.Join(root, "app")
	if err := os.MkdirAll(checkout, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := h.Deploy(context.Background(), h.env(t), DeployInput{Name: "app", Content: "services: {b: c}"}); err != nil {
		t.Fatalf("second Deploy: %v", err)
	}

	if _, statErr := os.Stat(checkout); statErr != nil {
		t.Fatalf("redeploying a content stack must not touch anything else: %v", statErr)
	}
}

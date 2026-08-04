package stack

import (
	"context"
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
			h := newHarness(t, fakeDocker{}, fakeCompose{})

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
	h := newHarness(t, fakeDocker{}, fakeCompose{deployed: deployed})

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
	h := newHarness(t, fakeDocker{}, fakeCompose{
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
	h := newHarness(t, fakeDocker{}, fakeCompose{})

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

package gitrepo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

func TestCloneFailureKeepsTheWorkingCheckout(t *testing.T) {
	t.Parallel()

	unreachable := httptest.NewServer(http.HandlerFunc(http.NotFound))
	defer unreachable.Close()

	project := t.TempDir()
	dir := filepath.Join(project, "repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	compose := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(compose, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Clone(context.Background(), dir, Options{URL: unreachable.URL + "/acme/app.git"}); err == nil {
		t.Fatal("expected the clone to fail")
	}

	if _, err := os.Stat(compose); err != nil {
		t.Fatalf("a failed clone must leave the working checkout alone: %v", err)
	}

	entries, err := os.ReadDir(project)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".clone-") || strings.HasSuffix(e.Name(), ".previous") {
			t.Fatalf("a failed clone must not leave %q behind", e.Name())
		}
	}
}

func refsFixture() []*plumbing.Reference {
	return []*plumbing.Reference{
		plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main")),
		plumbing.NewHashReference(plumbing.NewBranchReferenceName("main"), plumbing.NewHash("1111111111111111111111111111111111111111")),
		plumbing.NewHashReference(plumbing.NewBranchReferenceName("dev"), plumbing.NewHash("2222222222222222222222222222222222222222")),
		plumbing.NewHashReference(plumbing.NewTagReferenceName("v1.0"), plumbing.NewHash("3333333333333333333333333333333333333333")),
	}
}

func TestResolveRefDefaultBranchFollowsHeadSymref(t *testing.T) {
	got, err := resolveRef(refsFixture(), "")
	if err != nil {
		t.Fatalf("resolveRef: %v", err)
	}
	if got != "1111111111111111111111111111111111111111" {
		t.Fatalf("default branch: got %q", got)
	}
}

func TestResolveRefBranchAndTag(t *testing.T) {
	cases := map[string]string{
		"dev":  "2222222222222222222222222222222222222222",
		"main": "1111111111111111111111111111111111111111",
		"v1.0": "3333333333333333333333333333333333333333",
	}
	for ref, want := range cases {
		got, err := resolveRef(refsFixture(), ref)
		if err != nil {
			t.Errorf("resolveRef(%q): %v", ref, err)
			continue
		}
		if got != want {
			t.Errorf("resolveRef(%q) = %q, want %q", ref, got, want)
		}
	}
}

func TestResolveRefUnknown(t *testing.T) {
	if _, err := resolveRef(refsFixture(), "nope"); !errors.Is(err, ErrRef) {
		t.Fatalf("unknown ref: want ErrRef, got %v", err)
	}
}

func TestResolveRefRejectsZeroHead(t *testing.T) {
	refs := []*plumbing.Reference{
		plumbing.NewHashReference(plumbing.HEAD, plumbing.ZeroHash),
	}
	if _, err := resolveRef(refs, ""); !errors.Is(err, ErrRef) {
		t.Fatalf("zero head: want ErrRef, got %v", err)
	}
}

func TestComposePathAccepts(t *testing.T) {
	cases := map[string]string{
		"docker-compose.yml":       "docker-compose.yml",
		"./docker-compose.yml":     "docker-compose.yml",
		"deploy/compose.yaml":      "deploy/compose.yaml",
		"deploy/./prod/stack.yml":  "deploy/prod/stack.yml",
		"deploy/prod/../stack.yml": "deploy/stack.yml",
		"  docker-compose.yml  ":   "docker-compose.yml",
	}
	for in, want := range cases {
		got, err := ComposePath(in)
		if err != nil {
			t.Errorf("ComposePath(%q): unexpected error %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ComposePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestComposePathRejectsEscapes(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"..",
		"../compose.yml",
		"../../etc/passwd",
		"deploy/../../../etc/passwd",
		"/etc/passwd",
		"/docker-compose.yml",
	}
	for _, in := range cases {
		if got, err := ComposePath(in); err == nil {
			t.Errorf("ComposePath(%q) = %q, want error", in, got)
		}
	}
}

func TestNormalizeURLAccepts(t *testing.T) {
	for _, in := range []string{
		"https://github.com/acme/app",
		"https://github.com/acme/app.git",
		"http://gitea.internal:3000/acme/app.git",
	} {
		if _, err := NormalizeURL(in); err != nil {
			t.Errorf("NormalizeURL(%q): unexpected error %v", in, err)
		}
	}
}

func TestNormalizeURLRejectsCredentials(t *testing.T) {
	t.Parallel()

	for _, in := range []string{
		"https://bob:ghp_secret@github.com/acme/app.git",
		"https://oauth2:glpat-secret@gitlab.com/acme/app.git",
		"http://user:pass@gitea.internal:3000/acme/app.git",
		"https://:token@github.com/acme/app.git",
	} {
		got, err := NormalizeURL(in)
		if !errors.Is(err, ErrCredentialsInURL) {
			t.Errorf("NormalizeURL(%q) = %q, %v; want ErrCredentialsInURL", in, got, err)
		}
		if got != "" {
			t.Errorf("NormalizeURL(%q) leaked %q on rejection", in, got)
		}
	}
}

func TestNormalizeURLStripsBareUsername(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"https://bob@github.com/acme/app.git": "https://github.com/acme/app.git",
		"https://github.com/acme/app.git":     "https://github.com/acme/app.git",
		"http://gitea.internal:3000/a/b.git":  "http://gitea.internal:3000/a/b.git",
	}
	for in, want := range cases {
		got, err := NormalizeURL(in)
		if err != nil {
			t.Errorf("NormalizeURL(%q): unexpected error %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeURLRejectsNonHTTP(t *testing.T) {
	cases := []string{
		"",
		"file:///etc/passwd",
		"file:///Users/someone/secret-repo",
		"ssh://git@github.com/acme/app.git",
		"git://github.com/acme/app.git",
		"git@github.com:acme/app.git",
		"https://",
	}
	for _, in := range cases {
		if _, err := NormalizeURL(in); err == nil {
			t.Errorf("NormalizeURL(%q): want error, got nil", in)
		}
	}
}

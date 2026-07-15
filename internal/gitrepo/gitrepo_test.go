package gitrepo

import "testing"

func TestComposePathAccepts(t *testing.T) {
	cases := map[string]string{
		"docker-compose.yml":        "docker-compose.yml",
		"./docker-compose.yml":      "docker-compose.yml",
		"deploy/compose.yaml":       "deploy/compose.yaml",
		"deploy/./prod/stack.yml":   "deploy/prod/stack.yml",
		"deploy/prod/../stack.yml":  "deploy/stack.yml",
		"  docker-compose.yml  ":    "docker-compose.yml",
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

package stack

import (
	"context"
	"strings"
	"testing"
)

func TestEnvFileContentQuotesEveryValue(t *testing.T) {
	cases := map[string]struct {
		in   []EnvVar
		want string
	}{
		"plain value": {
			[]EnvVar{{Key: "PORT", Value: "8080"}},
			"PORT=\"8080\"\n",
		},
		"value with spaces": {
			[]EnvVar{{Key: "GREETING", Value: "hello world"}},
			"GREETING=\"hello world\"\n",
		},
		"empty key is skipped": {
			[]EnvVar{{Key: "  ", Value: "ignored"}, {Key: "KEEP", Value: "1"}},
			"KEEP=\"1\"\n",
		},
		"newline cannot inject a line": {
			[]EnvVar{{Key: "TOKEN", Value: "abc\nADMIN=true"}},
			"TOKEN=\"abc\\nADMIN=true\"\n",
		},
		"carriage return is escaped": {
			[]EnvVar{{Key: "TOKEN", Value: "abc\r\nX=1"}},
			"TOKEN=\"abc\\r\\nX=1\"\n",
		},
		"quotes are escaped": {
			[]EnvVar{{Key: "JSON", Value: `{"a":1}`}},
			"JSON=\"{\\\"a\\\":1}\"\n",
		},
		"backslash is escaped": {
			[]EnvVar{{Key: "PATH_LIKE", Value: `C:\tmp`}},
			"PATH_LIKE=\"C:\\\\tmp\"\n",
		},
		"a closing quote cannot escape the value": {
			[]EnvVar{{Key: "EVIL", Value: `"` + "\n" + `ADMIN=true`}},
			"EVIL=\"\\\"\\nADMIN=true\"\n",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := EnvFileContent(tc.in); got != tc.want {
				t.Errorf("EnvFileContent = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEnvFileContentNeverEmitsMoreLinesThanKeys(t *testing.T) {
	content := EnvFileContent([]EnvVar{
		{Key: "A", Value: "1\nB=2\nC=3"},
		{Key: "D", Value: "4"},
	})

	lines := strings.Count(strings.TrimSuffix(content, "\n"), "\n") + 1
	if lines != 2 {
		t.Fatalf("two variables must produce two lines, got %d:\n%s", lines, content)
	}
}

func TestDeployRejectsMalformedEnvKeys(t *testing.T) {
	for _, key := range []string{"MY-VAR", "1START", "has space", "a=b", "DOLLAR$", "quote\"key"} {
		t.Run(key, func(t *testing.T) {
			h := newHarness(t, fakeDocker{}.factory(), fakeCompose{})

			err := h.Deploy(context.Background(), h.env(t), DeployInput{
				Name:    "app",
				Content: "services: {}",
				Env:     []EnvVar{{Key: key, Value: "x"}},
			})
			if err == nil {
				t.Fatalf("key %q must be rejected", key)
			}
			if got := kindOf(t, err); got != KindInvalid {
				t.Fatalf("kind = %v, want KindInvalid", got)
			}
		})
	}
}

func TestDeployAcceptsValidEnvKeys(t *testing.T) {
	h := newHarness(t, fakeDocker{}.factory(), fakeCompose{})

	err := h.Deploy(context.Background(), h.env(t), DeployInput{
		Name:    "app",
		Content: "services: {}",
		Env: []EnvVar{
			{Key: "PORT", Value: "8080"},
			{Key: "_PRIVATE", Value: "x"},
			{Key: "DB_URL_2", Value: "y"},
			{Key: "  ", Value: "blank rows come from the form"},
		},
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
}

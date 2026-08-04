package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

const (
	headA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	headB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func pktLine(payload string) string {
	return fmt.Sprintf("%04x%s", len(payload)+4, payload)
}

func fakeRemote(t *testing.T, head string) string {
	t.Helper()
	body := pktLine("# service=git-upload-pack\n") + "0000" +
		pktLine(head+" HEAD\x00symref=HEAD:refs/heads/main\n") +
		pktLine(head+" refs/heads/main\n") + "0000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/acme/app.git"
}

func seedGitStack(t *testing.T, srv *Server, name, url, remoteHash string, interval int64) db.Stack {
	t.Helper()
	stack, err := srv.queries.UpsertStack(context.Background(), db.UpsertStackParams{
		EnvID:           1,
		Name:            name,
		Content:         "services: {}\n",
		Env:             "[]",
		CreatedBy:       "tester",
		UpdatedBy:       "tester",
		Source:          sourceGit,
		GitUrl:          url,
		GitRef:          "main",
		GitPath:         "docker-compose.yml",
		GitCommit:       remoteHash,
		GitRemoteHash:   remoteHash,
		GitAutoUpdate:   1,
		GitPollInterval: interval,
	})
	if err != nil {
		t.Fatalf("UpsertStack(%s): %v", name, err)
	}
	return stack
}

func reloadStack(t *testing.T, srv *Server, name string) db.Stack {
	t.Helper()
	stack, err := srv.queries.GetStack(context.Background(), db.GetStackParams{EnvID: 1, Name: name})
	if err != nil {
		t.Fatalf("GetStack(%s): %v", name, err)
	}
	return stack
}

func drainPoller(t *testing.T, srv *Server) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Wait(ctx); err != nil {
		t.Fatalf("git poller work did not finish: %v", err)
	}
}

func TestPollGitStacksLeavesAStoppedStackAlone(t *testing.T) {
	deployed := make(chan string, 1)
	srv := newTestServer(t)
	srv.docker = fakeDocker{stacks: []docker.Stack{{Name: "app", Running: 0, Total: 2, State: "stopped"}}}
	srv.compose = fakeCompose{deployedRepo: deployed}
	seedEnvironment(t, srv)
	seedGitStack(t, srv, "app", fakeRemote(t, headB), headA, 15)

	srv.pollGitStacks(context.Background())
	drainPoller(t, srv)

	select {
	case project := <-deployed:
		t.Fatalf("a stopped stack must not be redeployed, but %q was", project)
	default:
	}

	stack := reloadStack(t, srv, "app")
	if stack.GitRemoteHash != headB {
		t.Errorf("the new remote hash must still be recorded: got %q, want %q", stack.GitRemoteHash, headB)
	}
	if stack.GitLastError != "" {
		t.Errorf("skipping a stopped stack is not an error, got %q", stack.GitLastError)
	}
	if stack.GitCommit != headA {
		t.Errorf("a stopped stack must keep its deployed commit: got %q", stack.GitCommit)
	}
}

func TestPollGitStacksDoesNothingWhenTheRemoteIsUnchanged(t *testing.T) {
	deployed := make(chan string, 1)
	srv := newTestServer(t)
	srv.docker = fakeDocker{stacks: []docker.Stack{{Name: "app", Running: 2, Total: 2, State: "running"}}}
	srv.compose = fakeCompose{deployedRepo: deployed}
	seedEnvironment(t, srv)
	seedGitStack(t, srv, "app", fakeRemote(t, headA), headA, 15)

	srv.pollGitStacks(context.Background())
	drainPoller(t, srv)

	select {
	case project := <-deployed:
		t.Fatalf("an unchanged remote must not trigger a redeploy, but %q was deployed", project)
	default:
	}

	stack := reloadStack(t, srv, "app")
	if stack.GitLastCheckedAt == 0 {
		t.Error("an unchanged remote must still mark the stack as checked")
	}
	if stack.GitLastError != "" {
		t.Errorf("an unchanged remote is not an error, got %q", stack.GitLastError)
	}
}

func TestPollGitStacksHonoursThePollInterval(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	srv.compose = fakeCompose{}
	seedEnvironment(t, srv)

	fresh := seedGitStack(t, srv, "fresh", fakeRemote(t, headB), headA, 3600)
	seedGitStack(t, srv, "stale", fakeRemote(t, headB), headA, 15)

	if err := srv.queries.MarkStackChecked(context.Background(), db.MarkStackCheckedParams{
		ID:            fresh.ID,
		GitRemoteHash: headA,
	}); err != nil {
		t.Fatalf("MarkStackChecked: %v", err)
	}

	srv.pollGitStacks(context.Background())
	drainPoller(t, srv)

	if got := reloadStack(t, srv, "fresh").GitRemoteHash; got != headA {
		t.Errorf("a stack checked within its interval must be skipped, but its hash moved to %q", got)
	}
	if got := reloadStack(t, srv, "stale").GitRemoteHash; got != headB {
		t.Errorf("a stack past its interval must be checked: hash is %q, want %q", got, headB)
	}
}

func TestPollGitStacksAppliesTheMinimumInterval(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	srv.compose = fakeCompose{}
	seedEnvironment(t, srv)

	stack := seedGitStack(t, srv, "app", fakeRemote(t, headB), headA, 0)
	if err := srv.queries.MarkStackChecked(context.Background(), db.MarkStackCheckedParams{
		ID:            stack.ID,
		GitRemoteHash: headA,
	}); err != nil {
		t.Fatalf("MarkStackChecked: %v", err)
	}

	srv.pollGitStacks(context.Background())
	drainPoller(t, srv)

	if got := reloadStack(t, srv, "app").GitRemoteHash; got != headA {
		t.Errorf("an interval below the floor must still be throttled, but the hash moved to %q", got)
	}
}

func TestPollGitStacksSkipsAStackAlreadyInFlight(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	srv.compose = fakeCompose{}
	seedEnvironment(t, srv)

	stack := seedGitStack(t, srv, "app", fakeRemote(t, headB), headA, 15)
	if !srv.acquireGitStack(stack.ID) {
		t.Fatal("acquireGitStack on a free stack must succeed")
	}

	srv.pollGitStacks(context.Background())
	drainPoller(t, srv)

	if got := reloadStack(t, srv, "app").GitRemoteHash; got != headA {
		t.Errorf("a stack already in flight must be skipped, but its hash moved to %q", got)
	}

	srv.releaseGitStack(stack.ID)
	if !srv.acquireGitStack(stack.ID) {
		t.Fatal("releaseGitStack must make the stack available again")
	}
}

func TestPollGitStacksIgnoresStacksWithoutAutoUpdate(t *testing.T) {
	srv := newTestServer(t)
	srv.docker = fakeDocker{}
	srv.compose = fakeCompose{}
	seedEnvironment(t, srv)

	if _, err := srv.queries.UpsertStack(context.Background(), db.UpsertStackParams{
		EnvID: 1, Name: "manual", Content: "services: {}\n", Env: "[]",
		CreatedBy: "tester", UpdatedBy: "tester", Source: sourceGit,
		GitUrl: fakeRemote(t, headB), GitRef: "main", GitPath: "docker-compose.yml",
		GitCommit: headA, GitRemoteHash: headA, GitAutoUpdate: 0, GitPollInterval: 15,
	}); err != nil {
		t.Fatalf("UpsertStack: %v", err)
	}

	srv.pollGitStacks(context.Background())
	drainPoller(t, srv)

	if got := reloadStack(t, srv, "manual").GitRemoteHash; got != headA {
		t.Errorf("a stack without auto update must never be polled, but its hash moved to %q", got)
	}
}

func TestStackIsRunning(t *testing.T) {
	cases := map[string]struct {
		stacks  []docker.Stack
		want    bool
		wantErr bool
	}{
		"fully running":   {[]docker.Stack{{Name: "app", Running: 2, Total: 2}}, true, false},
		"partially up":    {[]docker.Stack{{Name: "app", Running: 1, Total: 2}}, true, false},
		"stopped":         {[]docker.Stack{{Name: "app", Running: 0, Total: 2}}, false, false},
		"not deployed":    {[]docker.Stack{{Name: "other", Running: 2, Total: 2}}, false, false},
		"no stack at all": {nil, false, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := newTestServer(t)
			srv.docker = fakeDocker{stacks: tc.stacks}
			seedEnvironment(t, srv)

			env, err := srv.queries.GetEnvironment(context.Background(), 1)
			if err != nil {
				t.Fatalf("GetEnvironment: %v", err)
			}

			got, err := srv.stackIsRunning(context.Background(), env, "app")
			if (err != nil) != tc.wantErr {
				t.Fatalf("stackIsRunning error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("stackIsRunning = %v, want %v", got, tc.want)
			}
		})
	}
}

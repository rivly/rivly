package stack

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/rivly/rivly/internal/database"
	"github.com/rivly/rivly/internal/database/db"
	"github.com/rivly/rivly/internal/docker"
)

const (
	headA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	headB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

type fakeDocker struct {
	stacks []docker.Stack
	err    error
}

func (f fakeDocker) Stacks(_ context.Context, _ int64, _ string) ([]docker.Stack, error) {
	return f.stacks, f.err
}

type fakeCompose struct {
	deployed chan string
	err      error
}

func (f fakeCompose) RepoDir(_ int64, project string) string {
	return filepath.Join("/tmp/rivly-test", project)
}

func (f fakeCompose) DeployRepo(_ context.Context, _ string, _ int64, project, _, _ string) (string, error) {
	if f.deployed != nil {
		select {
		case f.deployed <- project:
		default:
		}
	}
	return "", f.err
}

type fakeCredentials struct {
	err error
}

func (f fakeCredentials) Credentials(_ context.Context, _ int64) (string, string, error) {
	return "git", "token", f.err
}

type harness struct {
	*Service
	queries *db.Queries
	running sync.WaitGroup
}

func newHarness(t *testing.T, dockerClient Docker, composeRunner Compose) *harness {
	t.Helper()

	sqlDB, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.Migrate(sqlDB); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	queries := db.New(sqlDB)
	if _, err := queries.CreateEnvironment(context.Background(), db.CreateEnvironmentParams{
		Name: "local", Kind: "local", Url: "unix:///var/run/docker.sock",
	}); err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	h := &harness{queries: queries}
	h.Service = New(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		queries,
		dockerClient,
		composeRunner,
		fakeCredentials{},
		func(context.Context, db.Environment) {},
		func(fn func()) {
			h.running.Add(1)
			go func() {
				defer h.running.Done()
				fn()
			}()
		},
	)
	return h
}

func (h *harness) syncAndWait(t *testing.T) {
	t.Helper()
	h.Sync(context.Background())
	h.running.Wait()
}

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

func (h *harness) seed(t *testing.T, name, url, remoteHash string, interval int64, autoUpdate int64) db.Stack {
	t.Helper()
	st, err := h.queries.UpsertStack(context.Background(), db.UpsertStackParams{
		EnvID:           1,
		Name:            name,
		Content:         "services: {}\n",
		Env:             "[]",
		CreatedBy:       "tester",
		UpdatedBy:       "tester",
		Source:          SourceGit,
		GitUrl:          url,
		GitRef:          "main",
		GitPath:         "docker-compose.yml",
		GitCommit:       remoteHash,
		GitRemoteHash:   remoteHash,
		GitAutoUpdate:   autoUpdate,
		GitPollInterval: interval,
	})
	if err != nil {
		t.Fatalf("UpsertStack(%s): %v", name, err)
	}
	return st
}

func (h *harness) reload(t *testing.T, name string) db.Stack {
	t.Helper()
	st, err := h.queries.GetStack(context.Background(), db.GetStackParams{EnvID: 1, Name: name})
	if err != nil {
		t.Fatalf("GetStack(%s): %v", name, err)
	}
	return st
}

func (h *harness) markChecked(t *testing.T, id int64, hash string) {
	t.Helper()
	if err := h.queries.MarkStackChecked(context.Background(), db.MarkStackCheckedParams{
		ID: id, GitRemoteHash: hash,
	}); err != nil {
		t.Fatalf("MarkStackChecked: %v", err)
	}
}

func TestSyncLeavesAStoppedStackAlone(t *testing.T) {
	deployed := make(chan string, 1)
	h := newHarness(t,
		fakeDocker{stacks: []docker.Stack{{Name: "app", Running: 0, Total: 2}}},
		fakeCompose{deployed: deployed},
	)
	h.seed(t, "app", fakeRemote(t, headB), headA, 15, 1)

	h.syncAndWait(t)

	select {
	case project := <-deployed:
		t.Fatalf("a stopped stack must not be redeployed, but %q was", project)
	default:
	}

	st := h.reload(t, "app")
	if st.GitRemoteHash != headB {
		t.Errorf("the new remote hash must still be recorded: got %q, want %q", st.GitRemoteHash, headB)
	}
	if st.GitLastError != "" {
		t.Errorf("skipping a stopped stack is not an error, got %q", st.GitLastError)
	}
	if st.GitCommit != headA {
		t.Errorf("a stopped stack must keep its deployed commit, got %q", st.GitCommit)
	}
}

func TestSyncDoesNothingWhenTheRemoteIsUnchanged(t *testing.T) {
	deployed := make(chan string, 1)
	h := newHarness(t,
		fakeDocker{stacks: []docker.Stack{{Name: "app", Running: 2, Total: 2}}},
		fakeCompose{deployed: deployed},
	)
	h.seed(t, "app", fakeRemote(t, headA), headA, 15, 1)

	h.syncAndWait(t)

	select {
	case project := <-deployed:
		t.Fatalf("an unchanged remote must not trigger a redeploy, but %q was deployed", project)
	default:
	}

	st := h.reload(t, "app")
	if st.GitLastCheckedAt == 0 {
		t.Error("an unchanged remote must still mark the stack as checked")
	}
	if st.GitLastError != "" {
		t.Errorf("an unchanged remote is not an error, got %q", st.GitLastError)
	}
}

func TestSyncHonoursThePollInterval(t *testing.T) {
	h := newHarness(t, fakeDocker{}, fakeCompose{})

	fresh := h.seed(t, "fresh", fakeRemote(t, headB), headA, 3600, 1)
	h.seed(t, "stale", fakeRemote(t, headB), headA, 15, 1)
	h.markChecked(t, fresh.ID, headA)

	h.syncAndWait(t)

	if got := h.reload(t, "fresh").GitRemoteHash; got != headA {
		t.Errorf("a stack checked within its interval must be skipped, but its hash moved to %q", got)
	}
	if got := h.reload(t, "stale").GitRemoteHash; got != headB {
		t.Errorf("a stack past its interval must be checked: hash is %q, want %q", got, headB)
	}
}

func TestSyncAppliesTheMinimumInterval(t *testing.T) {
	h := newHarness(t, fakeDocker{}, fakeCompose{})

	st := h.seed(t, "app", fakeRemote(t, headB), headA, 0, 1)
	h.markChecked(t, st.ID, headA)

	h.syncAndWait(t)

	if got := h.reload(t, "app").GitRemoteHash; got != headA {
		t.Errorf("an interval below the floor must still be throttled, but the hash moved to %q", got)
	}
}

func TestSyncSkipsAStackAlreadyInFlight(t *testing.T) {
	h := newHarness(t, fakeDocker{}, fakeCompose{})

	st := h.seed(t, "app", fakeRemote(t, headB), headA, 15, 1)
	if !h.Acquire(st.ID) {
		t.Fatal("Acquire on a free stack must succeed")
	}

	h.syncAndWait(t)

	if got := h.reload(t, "app").GitRemoteHash; got != headA {
		t.Errorf("a stack already in flight must be skipped, but its hash moved to %q", got)
	}

	h.Release(st.ID)
	if !h.Acquire(st.ID) {
		t.Fatal("Release must make the stack available again")
	}
}

func TestSyncIgnoresStacksWithoutAutoUpdate(t *testing.T) {
	h := newHarness(t, fakeDocker{}, fakeCompose{})
	h.seed(t, "manual", fakeRemote(t, headB), headA, 15, 0)

	h.syncAndWait(t)

	if got := h.reload(t, "manual").GitRemoteHash; got != headA {
		t.Errorf("a stack without auto update must never be polled, but its hash moved to %q", got)
	}
}

func TestIsRunning(t *testing.T) {
	cases := map[string]struct {
		stacks []docker.Stack
		want   bool
	}{
		"fully running":   {[]docker.Stack{{Name: "app", Running: 2, Total: 2}}, true},
		"partially up":    {[]docker.Stack{{Name: "app", Running: 1, Total: 2}}, true},
		"stopped":         {[]docker.Stack{{Name: "app", Running: 0, Total: 2}}, false},
		"not deployed":    {[]docker.Stack{{Name: "other", Running: 2, Total: 2}}, false},
		"no stack at all": {nil, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, fakeDocker{stacks: tc.stacks}, fakeCompose{})

			env, err := h.queries.GetEnvironment(context.Background(), 1)
			if err != nil {
				t.Fatalf("GetEnvironment: %v", err)
			}

			got, err := h.isRunning(context.Background(), env, "app")
			if err != nil {
				t.Fatalf("isRunning: %v", err)
			}
			if got != tc.want {
				t.Errorf("isRunning = %v, want %v", got, tc.want)
			}
		})
	}
}

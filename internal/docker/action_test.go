package docker

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type call struct {
	method string
	path   string
}

func recordingEngine(t *testing.T, body string) (*Client, func() []call) {
	t.Helper()

	var mu sync.Mutex
	var calls []call

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/_ping") {
			mu.Lock()
			calls = append(calls, call{r.Method, r.URL.Path})
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)

	manager := NewManager(nil)
	t.Cleanup(manager.Close)

	client, err := manager.Client(1, ts.URL)
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	return client, func() []call {
		mu.Lock()
		defer mu.Unlock()
		return append([]call(nil), calls...)
	}
}

func TestContainerActionHitsTheRightEndpoint(t *testing.T) {
	t.Parallel()

	cases := map[string]call{
		"start":   {http.MethodPost, "/containers/abc/start"},
		"stop":    {http.MethodPost, "/containers/abc/stop"},
		"restart": {http.MethodPost, "/containers/abc/restart"},
		"pause":   {http.MethodPost, "/containers/abc/pause"},
		"unpause": {http.MethodPost, "/containers/abc/unpause"},
		"kill":    {http.MethodPost, "/containers/abc/kill"},
		"remove":  {http.MethodDelete, "/containers/abc"},
	}

	for action, want := range cases {
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			client, recorded := recordingEngine(t, `{}`)

			if err := client.ContainerAction(context.Background(), "abc", action); err != nil {
				t.Fatalf("ContainerAction(%q): %v", action, err)
			}

			calls := recorded()
			if len(calls) != 1 {
				t.Fatalf("want exactly one daemon call, got %+v", calls)
			}
			if calls[0].method != want.method || !strings.HasSuffix(calls[0].path, want.path) {
				t.Fatalf("%q called %s %s, want %s ...%s", action, calls[0].method, calls[0].path, want.method, want.path)
			}
		})
	}
}

func TestUnknownActionsNeverReachTheDaemon(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"", "destroy", "START", "rm"} {
		client, recorded := recordingEngine(t, `{}`)

		if err := client.ContainerAction(context.Background(), "abc", action); err == nil {
			t.Errorf("action %q must be rejected", action)
		}
		if calls := recorded(); len(calls) != 0 {
			t.Errorf("action %q reached the daemon: %+v", action, calls)
		}
	}
}

func TestResourceActionsHitTheRightEndpoint(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		run  func(*Client) error
		want call
	}{
		"image remove": {
			func(c *Client) error { return c.ImageAction(context.Background(), "img1", "remove") },
			call{http.MethodDelete, "/images/img1"},
		},
		"volume remove": {
			func(c *Client) error { return c.VolumeAction(context.Background(), "app_data", "remove") },
			call{http.MethodDelete, "/volumes/app_data"},
		},
		"network remove": {
			func(c *Client) error { return c.NetworkAction(context.Background(), "n1", "remove") },
			call{http.MethodDelete, "/networks/n1"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client, recorded := recordingEngine(t, `[]`)

			if err := tc.run(client); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			calls := recorded()
			if len(calls) != 1 {
				t.Fatalf("want one daemon call, got %+v", calls)
			}
			if calls[0].method != tc.want.method || !strings.HasSuffix(calls[0].path, tc.want.path) {
				t.Fatalf("called %s %s, want %s ...%s", calls[0].method, calls[0].path, tc.want.method, tc.want.path)
			}
		})
	}
}

func TestUnknownResourceActionsAreRejected(t *testing.T) {
	t.Parallel()

	client, _ := recordingEngine(t, `[]`)
	ctx := context.Background()

	if err := client.ImageAction(ctx, "img1", "prune"); err == nil {
		t.Error("image action must be limited to remove")
	}
	if err := client.VolumeAction(ctx, "v", "resize"); err == nil {
		t.Error("volume action must be limited to remove")
	}
	if err := client.NetworkAction(ctx, "n", "connect"); err == nil {
		t.Error("network action must be limited to remove")
	}
}

func TestStackActionAppliesToEveryContainerOfTheProject(t *testing.T) {
	t.Parallel()

	client, recorded := recordingEngine(t, twoStacksAndAnOrphan)

	if err := client.StackAction(context.Background(), "app", "stop"); err != nil {
		t.Fatalf("StackAction: %v", err)
	}

	var stopped []string
	for _, c := range recorded() {
		if strings.HasSuffix(c.path, "/stop") {
			stopped = append(stopped, c.path)
		}
	}
	if len(stopped) != 2 {
		t.Fatalf("the app project has two containers, got %d stop calls: %v", len(stopped), stopped)
	}
	for _, path := range stopped {
		if strings.Contains(path, "/b1/") || strings.Contains(path, "/z1/") {
			t.Fatalf("a stack action must not touch another project: %s", path)
		}
	}
}

func TestNetworkCreateRejectsABadSubnet(t *testing.T) {
	t.Parallel()

	client, recorded := recordingEngine(t, `{"Id":"n1"}`)

	if _, err := client.NetworkCreate(context.Background(), NetworkCreateInput{Name: "n", Subnet: "not-a-cidr"}); err == nil {
		t.Fatal("an invalid subnet must be rejected before the daemon is asked")
	}
	if calls := recorded(); len(calls) != 0 {
		t.Fatalf("nothing must reach the daemon: %+v", calls)
	}
}

func TestVolumeCreateDefaultsToTheLocalDriver(t *testing.T) {
	t.Parallel()

	var body string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/volumes/create") {
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			body = string(buf)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Name":"app_data","Driver":"local","CreatedAt":"2026-01-01T00:00:00Z"}`))
	}))
	defer ts.Close()

	manager := NewManager(nil)
	defer manager.Close()
	client, err := manager.Client(1, ts.URL)
	if err != nil {
		t.Fatalf("Client: %v", err)
	}

	if _, err := client.VolumeCreate(context.Background(), VolumeCreateInput{Name: "app_data"}); err != nil {
		t.Fatalf("VolumeCreate: %v", err)
	}
	if !strings.Contains(body, `"local"`) {
		t.Fatalf("an unspecified driver must default to local, sent %s", body)
	}
}

func TestClientIsIsolatedPerEnvironment(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil)
	defer manager.Close()

	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer first.Close()

	a, err := manager.Client(1, first.URL)
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	b, err := manager.Client(2, first.URL)
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	if fmt.Sprintf("%p", a) == fmt.Sprintf("%p", b) {
		t.Fatal("two environments must not share a handle")
	}
}

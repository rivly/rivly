package docker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeEngine(t *testing.T, routes map[string]string) *Client {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		for suffix, body := range routes {
			if strings.HasSuffix(r.URL.Path, suffix) {
				_, _ = w.Write([]byte(body))
				return
			}
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(ts.Close)

	manager := NewManager(nil)
	t.Cleanup(manager.Close)

	client, err := manager.Client(1, ts.URL)
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	return client
}

const twoStacksAndAnOrphan = `[
 {"Id":"a1","Names":["/app-web-1"],"Image":"nginx","State":"running","Labels":{"com.docker.compose.project":"app","com.docker.compose.service":"web","com.docker.compose.project.working_dir":"/srv/app"},"Mounts":[{"Name":"app_data"}],"NetworkSettings":{"Networks":{"app_default":{"IPAddress":"172.20.0.2"}}},"ImageID":"sha256:img1"},
 {"Id":"a2","Names":["/app-db-1"],"Image":"postgres","State":"exited","Labels":{"com.docker.compose.project":"app","com.docker.compose.service":"db"},"ImageID":"sha256:img2"},
 {"Id":"b1","Names":["/blog-web-1"],"Image":"ghost","State":"running","Labels":{"com.docker.compose.project":"blog","com.docker.compose.service":"web"},"ImageID":"sha256:img1"},
 {"Id":"z1","Names":["/standalone"],"Image":"redis","State":"running","Labels":{},"ImageID":"sha256:img3"}
]`

func TestStacksGroupsByComposeLabel(t *testing.T) {
	t.Parallel()

	client := fakeEngine(t, map[string]string{"/containers/json": twoStacksAndAnOrphan})

	stacks, err := client.Stacks(context.Background())
	if err != nil {
		t.Fatalf("Stacks: %v", err)
	}

	if len(stacks) != 2 {
		t.Fatalf("a container without a compose label is not a stack, got %+v", stacks)
	}

	byName := map[string]Stack{}
	for _, s := range stacks {
		byName[s.Name] = s
	}

	app := byName["app"]
	if app.Services != 2 || app.Total != 2 || app.Running != 1 {
		t.Errorf("app counts = %d services, %d/%d running: %+v", app.Services, app.Running, app.Total, app)
	}
	if app.State != "partial" {
		t.Errorf("one container up out of two is partial, got %q", app.State)
	}
	if app.WorkingDir != "/srv/app" {
		t.Errorf("the working dir label must be surfaced, got %q", app.WorkingDir)
	}
	if app.Type != "external" {
		t.Errorf("a discovered stack is external until Rivly claims it, got %q", app.Type)
	}

	if blog := byName["blog"]; blog.State != "running" {
		t.Errorf("every container up means running, got %q", blog.State)
	}
}

func TestStacksReportsAFullyStoppedProject(t *testing.T) {
	t.Parallel()

	client := fakeEngine(t, map[string]string{"/containers/json": `[
	 {"Id":"a1","Names":["/app-web-1"],"State":"exited","Labels":{"com.docker.compose.project":"app","com.docker.compose.service":"web"}}
	]`})

	stacks, err := client.Stacks(context.Background())
	if err != nil {
		t.Fatalf("Stacks: %v", err)
	}
	if len(stacks) != 1 || stacks[0].State != "stopped" {
		t.Fatalf("stacks = %+v, want a single stopped stack", stacks)
	}
}

func TestContainersTranslateNamesAndStacks(t *testing.T) {
	t.Parallel()

	client := fakeEngine(t, map[string]string{"/containers/json": twoStacksAndAnOrphan})

	containers, err := client.Containers(context.Background())
	if err != nil {
		t.Fatalf("Containers: %v", err)
	}
	if len(containers) != 4 {
		t.Fatalf("want every container, got %d", len(containers))
	}

	first := containers[0]
	if first.Name != "app-web-1" {
		t.Errorf("the leading slash must be stripped, got %q", first.Name)
	}
	if first.Stack != "app" {
		t.Errorf("stack = %q, want app", first.Stack)
	}
	if first.IP != "172.20.0.2" {
		t.Errorf("the first network address must be surfaced, got %q", first.IP)
	}
	if containers[3].Stack != "" {
		t.Errorf("a container without a compose label has no stack, got %q", containers[3].Stack)
	}
}

func TestImagesFlagTheOnesInUse(t *testing.T) {
	t.Parallel()

	client := fakeEngine(t, map[string]string{
		"/containers/json": twoStacksAndAnOrphan,
		"/images/json": `[
		 {"Id":"sha256:img1","RepoTags":["nginx:latest"],"Size":100,"Created":1700000000},
		 {"Id":"sha256:img9","RepoTags":["<none>:<none>"],"Size":50,"Created":1700000001}
		]`,
	})

	images, err := client.Images(context.Background())
	if err != nil {
		t.Fatalf("Images: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("want two images, got %d", len(images))
	}

	byID := map[string]Image{}
	for _, img := range images {
		byID[img.ID] = img
	}

	used := byID["img1"]
	if !used.InUse {
		t.Error("an image backing a running container must be flagged in use")
	}
	if len(used.Tags) != 1 || used.Tags[0] != "nginx:latest" {
		t.Errorf("tags = %v", used.Tags)
	}

	dangling := byID["img9"]
	if dangling.InUse {
		t.Error("an image no container references is not in use")
	}
	if len(dangling.Tags) != 0 {
		t.Errorf("the <none>:<none> placeholder is not a tag, got %v", dangling.Tags)
	}
}

func TestVolumesFlagTheOnesMounted(t *testing.T) {
	t.Parallel()

	client := fakeEngine(t, map[string]string{
		"/containers/json": twoStacksAndAnOrphan,
		"/volumes": `{"Volumes":[
		 {"Name":"app_data","Driver":"local","Mountpoint":"/var/lib/docker/volumes/app_data/_data","CreatedAt":"2026-01-01T00:00:00Z","Labels":{"com.docker.compose.project":"app"}},
		 {"Name":"orphan","Driver":"local","Mountpoint":"/var/lib/docker/volumes/orphan/_data","CreatedAt":"2026-01-01T00:00:00Z"}
		]}`,
	})

	volumes, err := client.Volumes(context.Background())
	if err != nil {
		t.Fatalf("Volumes: %v", err)
	}
	if len(volumes) != 2 {
		t.Fatalf("want two volumes, got %d", len(volumes))
	}

	for _, v := range volumes {
		switch v.Name {
		case "app_data":
			if !v.InUse {
				t.Error("a mounted volume must be flagged in use")
			}
			if v.Stack != "app" {
				t.Errorf("the compose label must be surfaced, got %q", v.Stack)
			}
		case "orphan":
			if v.InUse {
				t.Error("an unmounted volume must not be flagged in use, that is what makes pruning safe")
			}
		}
	}
}

func TestNetworksProtectThePredefinedOnes(t *testing.T) {
	t.Parallel()

	client := fakeEngine(t, map[string]string{
		"/containers/json": twoStacksAndAnOrphan,
		"/networks": `[
		 {"Id":"n1","Name":"app_default","Driver":"bridge","Scope":"local","Created":"2026-01-01T00:00:00Z","Labels":{"com.docker.compose.project":"app"}},
		 {"Id":"n2","Name":"unused_net","Driver":"bridge","Scope":"local","Created":"2026-01-01T00:00:00Z"},
		 {"Id":"n3","Name":"bridge","Driver":"bridge","Scope":"local","Created":"2026-01-01T00:00:00Z"},
		 {"Id":"n4","Name":"host","Driver":"host","Scope":"local","Created":"2026-01-01T00:00:00Z"}
		]`,
	})

	networks, err := client.Networks(context.Background())
	if err != nil {
		t.Fatalf("Networks: %v", err)
	}

	inUse := map[string]bool{}
	for _, n := range networks {
		inUse[n.Name] = n.InUse
	}

	if !inUse["app_default"] {
		t.Error("a network a container is attached to must be in use")
	}
	if inUse["unused_net"] {
		t.Error("an unattached network must not be in use")
	}
	if !inUse["bridge"] || !inUse["host"] {
		t.Error("the predefined networks must always count as in use, deleting them breaks the daemon")
	}
}

func TestClientSurfacesADaemonFailure(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"daemon is unwell"}`))
	}))
	defer ts.Close()

	manager := NewManager(nil)
	defer manager.Close()
	client, err := manager.Client(1, ts.URL)
	if err != nil {
		t.Fatalf("Client: %v", err)
	}

	if _, err := client.Containers(context.Background()); err == nil {
		t.Fatal("a daemon error must reach the caller, not be swallowed into an empty list")
	}
}

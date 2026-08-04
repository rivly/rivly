package docker

import (
	"context"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
)

func TestComputeStatsCPUPercent(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		stats container.StatsResponse
		want  float64
	}{
		"half of two cpus": {
			container.StatsResponse{
				CPUStats:    container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 200}, SystemUsage: 400, OnlineCPUs: 2},
				PreCPUStats: container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 100}, SystemUsage: 200},
			},
			100,
		},
		"no system delta means unknown, not infinite": {
			container.StatsResponse{
				CPUStats:    container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 200}, SystemUsage: 400, OnlineCPUs: 2},
				PreCPUStats: container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 100}, SystemUsage: 400},
			},
			0,
		},
		"idle container": {
			container.StatsResponse{
				CPUStats:    container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 100}, SystemUsage: 400, OnlineCPUs: 4},
				PreCPUStats: container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 100}, SystemUsage: 200},
			},
			0,
		},
		"cpu count falls back to the per-cpu list": {
			container.StatsResponse{
				CPUStats: container.CPUStats{
					CPUUsage:    container.CPUUsage{TotalUsage: 200, PercpuUsage: []uint64{50, 50, 50, 50}},
					SystemUsage: 400,
				},
				PreCPUStats: container.CPUStats{CPUUsage: container.CPUUsage{TotalUsage: 100}, SystemUsage: 200},
			},
			200,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := computeStats(tc.stats).CPUPercent; got != tc.want {
				t.Fatalf("CPUPercent = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestComputeStatsMemoryExcludesCache(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		stats     container.StatsResponse
		wantUsage uint64
		wantPct   float64
	}{
		"inactive file is not real usage": {
			container.StatsResponse{MemoryStats: container.MemoryStats{
				Usage: 1000, Limit: 2000,
				Stats: map[string]uint64{"inactive_file": 400},
			}},
			600, 30,
		},
		"cgroup v1 falls back to cache": {
			container.StatsResponse{MemoryStats: container.MemoryStats{
				Usage: 1000, Limit: 2000,
				Stats: map[string]uint64{"cache": 500},
			}},
			500, 25,
		},
		"a cache larger than usage is ignored": {
			container.StatsResponse{MemoryStats: container.MemoryStats{
				Usage: 100, Limit: 1000,
				Stats: map[string]uint64{"inactive_file": 900},
			}},
			100, 10,
		},
		"no limit means no percentage, not a division by zero": {
			container.StatsResponse{MemoryStats: container.MemoryStats{Usage: 500}},
			500, 0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := computeStats(tc.stats)
			if got.MemUsage != tc.wantUsage {
				t.Errorf("MemUsage = %d, want %d", got.MemUsage, tc.wantUsage)
			}
			if got.MemPercent != tc.wantPct {
				t.Errorf("MemPercent = %v, want %v", got.MemPercent, tc.wantPct)
			}
		})
	}
}

func TestComputeStatsSumsInterfacesAndDevices(t *testing.T) {
	t.Parallel()

	got := computeStats(container.StatsResponse{
		Networks: map[string]container.NetworkStats{
			"eth0": {RxBytes: 100, TxBytes: 10},
			"eth1": {RxBytes: 200, TxBytes: 20},
		},
		BlkioStats: container.BlkioStats{IoServiceBytesRecursive: []container.BlkioStatEntry{
			{Op: "Read", Value: 1000},
			{Op: "read", Value: 500},
			{Op: "Write", Value: 300},
			{Op: "Sync", Value: 999},
		}},
		PidsStats: container.PidsStats{Current: 7},
	})

	if got.NetRx != 300 || got.NetTx != 30 {
		t.Errorf("network totals = %d/%d, want 300/30", got.NetRx, got.NetTx)
	}
	if got.BlockRead != 1500 {
		t.Errorf("BlockRead = %d, want 1500, operation names differ in case across drivers", got.BlockRead)
	}
	if got.BlockWrite != 300 {
		t.Errorf("BlockWrite = %d, want 300", got.BlockWrite)
	}
	if got.Pids != 7 {
		t.Errorf("Pids = %d, want 7", got.Pids)
	}
}

func TestEventIsMeaningful(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		msg  events.Message
		want bool
	}{
		"container start":      {events.Message{Type: events.ContainerEventType, Action: "start"}, true},
		"container die":        {events.Message{Type: events.ContainerEventType, Action: "die"}, true},
		"exec noise":           {events.Message{Type: events.ContainerEventType, Action: "exec_start: sh"}, false},
		"health check noise":   {events.Message{Type: events.ContainerEventType, Action: "health_status: healthy"}, false},
		"terminal resize":      {events.Message{Type: events.ContainerEventType, Action: "resize"}, false},
		"attach":               {events.Message{Type: events.ContainerEventType, Action: "attach"}, false},
		"image pull":           {events.Message{Type: events.ImageEventType, Action: "pull"}, true},
		"volume create":        {events.Message{Type: events.VolumeEventType, Action: "create"}, true},
		"network connect":      {events.Message{Type: events.NetworkEventType, Action: "connect"}, true},
		"daemon event ignored": {events.Message{Type: events.DaemonEventType, Action: "reload"}, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := eventIsMeaningful(tc.msg); got != tc.want {
				t.Fatalf("eventIsMeaningful = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTranslatePorts(t *testing.T) {
	t.Parallel()

	exposed, bindings, err := translatePorts([]PortMapping{
		{ContainerPort: "80", HostPort: "8080"},
		{ContainerPort: "53", Proto: "udp"},
	})
	if err != nil {
		t.Fatalf("translatePorts: %v", err)
	}

	if len(exposed) != 2 {
		t.Fatalf("exposed = %v, want two ports", exposed)
	}
	if len(bindings) != 1 {
		t.Fatalf("a port without a host port must not be published: %v", bindings)
	}
	for port, binds := range bindings {
		if port.Port() != "80" || binds[0].HostPort != "8080" {
			t.Errorf("binding = %v -> %v", port, binds)
		}
	}
}

func TestTranslatePortsRejectsGarbage(t *testing.T) {
	t.Parallel()

	for _, port := range []string{"", "http", "99999", "-1"} {
		if _, _, err := translatePorts([]PortMapping{{ContainerPort: port}}); err == nil {
			t.Errorf("container port %q must be rejected", port)
		}
	}
}

func TestTranslateMountsInfersTheType(t *testing.T) {
	t.Parallel()

	got := translateMounts([]MountInput{
		{Source: "app_data", Target: "/data"},
		{Source: "/srv/app", Target: "/app"},
		{Source: "./local", Target: "/local", ReadOnly: true},
	})

	if len(got) != 3 {
		t.Fatalf("want three mounts, got %d", len(got))
	}
	if got[0].Type != mount.TypeVolume {
		t.Errorf("a bare name is a named volume, got %v", got[0].Type)
	}
	if got[1].Type != mount.TypeBind {
		t.Errorf("an absolute path is a bind mount, got %v", got[1].Type)
	}
	if got[2].Type != mount.TypeBind || !got[2].ReadOnly {
		t.Errorf("a relative path is a bind mount and keeps read-only: %+v", got[2])
	}
}

func TestBoundPortsKeepsUnpublishedPorts(t *testing.T) {
	t.Parallel()

	published, err := network.ParsePort("80/tcp")
	if err != nil {
		t.Fatalf("ParsePort: %v", err)
	}
	silent, err := network.ParsePort("9000/tcp")
	if err != nil {
		t.Fatalf("ParsePort: %v", err)
	}

	got := boundPorts(network.PortMap{
		published: {{HostPort: "8080"}},
		silent:    nil,
	})

	if len(got) != 2 {
		t.Fatalf("an exposed but unpublished port must still be reported, got %+v", got)
	}
	for _, p := range got {
		if p.PrivatePort == 9000 && p.PublicPort != 0 {
			t.Errorf("an unpublished port has no public port: %+v", p)
		}
		if p.PrivatePort == 80 && p.PublicPort != 8080 {
			t.Errorf("published port lost its host port: %+v", p)
		}
	}
}

func TestAttachmentsToleratesANilEndpoint(t *testing.T) {
	t.Parallel()

	got := attachments(map[string]*network.EndpointSettings{"bridge": nil})
	if len(got) != 1 || got[0].Name != "bridge" || got[0].IP != "" {
		t.Fatalf("a network with no endpoint settings must still be listed: %+v", got)
	}
}

func TestLogWriterSplitsLines(t *testing.T) {
	t.Parallel()

	out := make(chan LogLine, 8)
	w := &logWriter{ctx: context.Background(), out: out, stream: "stdout"}

	if _, err := w.Write([]byte("first\r\nsecond\npar")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := w.Write([]byte("tial\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	w.flush()
	close(out)

	var got []string
	for line := range out {
		if line.Stream != "stdout" {
			t.Errorf("stream = %q", line.Stream)
		}
		got = append(got, line.Message)
	}

	want := []string{"first", "second", "partial"}
	if len(got) != len(want) {
		t.Fatalf("lines = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLogWriterStopsOnCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	w := &logWriter{ctx: ctx, out: make(chan LogLine), stream: "stdout"}
	if _, err := w.Write([]byte("dropped\n")); err == nil {
		t.Fatal("a cancelled stream must stop the copy instead of blocking forever")
	}
}

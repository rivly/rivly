package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

const actionTimeout = 60 * time.Second

var execShell = []string{"/bin/sh", "-c", "exec $(command -v bash || command -v sh)"}

const callTimeout = 5 * time.Second

const (
	composeProjectLabel    = "com.docker.compose.project"
	composeServiceLabel    = "com.docker.compose.service"
	composeWorkingDirLabel = "com.docker.compose.project.working_dir"
)

type Manager struct {
	mu      sync.Mutex
	clients map[int64]*client.Client
}

func NewManager() *Manager {
	return &Manager{clients: make(map[int64]*client.Client)}
}

type Status struct {
	Up         bool
	APIVersion string
	Error      string
}

type SystemInfo struct {
	ServerVersion     string
	OSType            string
	Architecture      string
	KernelVersion     string
	OperatingSystem   string
	Name              string
	Swarm             bool
	Nodes             int
	NCPU              int
	MemTotal          int64
	Containers        int
	ContainersRunning int
	ContainersPaused  int
	ContainersStopped int
	Images            int
}

func (m *Manager) clientFor(id int64, host string) (*client.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clients[id]; ok {
		return c, nil
	}
	c, err := client.New(client.WithHost(host))
	if err != nil {
		return nil, err
	}
	m.clients[id] = c
	return c, nil
}

func (m *Manager) Ping(ctx context.Context, id int64, host string) Status {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return Status{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		return Status{Error: err.Error()}
	}
	return Status{Up: true, APIVersion: res.APIVersion}
}

func (m *Manager) Info(ctx context.Context, id int64, host string) (SystemInfo, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return SystemInfo{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := cli.Info(ctx, client.InfoOptions{})
	if err != nil {
		return SystemInfo{}, err
	}
	i := res.Info
	return SystemInfo{
		ServerVersion:     i.ServerVersion,
		OSType:            i.OSType,
		Architecture:      i.Architecture,
		KernelVersion:     i.KernelVersion,
		OperatingSystem:   i.OperatingSystem,
		Name:              i.Name,
		Swarm:             i.Swarm.LocalNodeState == "active",
		Nodes:             i.Swarm.Nodes,
		NCPU:              i.NCPU,
		MemTotal:          i.MemTotal,
		Containers:        i.Containers,
		ContainersRunning: i.ContainersRunning,
		ContainersPaused:  i.ContainersPaused,
		ContainersStopped: i.ContainersStopped,
		Images:            i.Images,
	}, nil
}

type Container struct {
	ID      string
	Name    string
	Image   string
	State   string
	Status  string
	Stack   string
	Created int64
	IP      string
	Ports   []Port
}

type Port struct {
	PrivatePort uint16
	PublicPort  uint16
	Type        string
	IP          string
}

func (m *Manager) Containers(ctx context.Context, id int64, host string) ([]Container, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	out := make([]Container, 0, len(res.Items))
	for _, c := range res.Items {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		ports := make([]Port, 0, len(c.Ports))
		for _, p := range c.Ports {
			ip := ""
			if p.IP.IsValid() {
				ip = p.IP.String()
			}
			ports = append(ports, Port{
				PrivatePort: p.PrivatePort,
				PublicPort:  p.PublicPort,
				Type:        p.Type,
				IP:          ip,
			})
		}
		ip := ""
		if c.NetworkSettings != nil {
			for _, n := range c.NetworkSettings.Networks {
				if n != nil && n.IPAddress.IsValid() {
					ip = n.IPAddress.String()
					break
				}
			}
		}
		out = append(out, Container{
			ID:      c.ID,
			Name:    name,
			Image:   c.Image,
			State:   string(c.State),
			Status:  c.Status,
			Stack:   c.Labels[composeProjectLabel],
			Created: c.Created,
			IP:      ip,
			Ports:   ports,
		})
	}
	return out, nil
}

type NetworkAttachment struct {
	Name string
	IP   string
}

type Mount struct {
	Type        string
	Source      string
	Destination string
	Name        string
	RW          bool
}

type ContainerDetail struct {
	ID            string
	Name          string
	Image         string
	State         string
	Created       int64
	StartedAt     string
	Command       string
	RestartPolicy string
	Ports         []Port
	Networks      []NetworkAttachment
	Mounts        []Mount
	Env           []string
	Labels        map[string]string
}

func (m *Manager) ContainerDetail(ctx context.Context, id int64, host, containerID string) (ContainerDetail, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return ContainerDetail{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return ContainerDetail{}, err
	}
	c := res.Container

	detail := ContainerDetail{
		ID:     c.ID,
		Name:   strings.TrimPrefix(c.Name, "/"),
		Image:  c.Image,
		Labels: map[string]string{},
	}
	if t, perr := time.Parse(time.RFC3339Nano, c.Created); perr == nil {
		detail.Created = t.Unix()
	}
	if c.State != nil {
		detail.State = string(c.State.Status)
		detail.StartedAt = c.State.StartedAt
	}
	if c.HostConfig != nil {
		detail.RestartPolicy = string(c.HostConfig.RestartPolicy.Name)
	}
	if c.Config != nil {
		detail.Image = c.Config.Image
		detail.Env = c.Config.Env
		detail.Labels = c.Config.Labels
	}
	detail.Command = strings.TrimSpace(c.Path + " " + strings.Join(c.Args, " "))

	if c.NetworkSettings != nil {
		for p, bindings := range c.NetworkSettings.Ports {
			priv, _ := strconv.ParseUint(p.Port(), 10, 16)
			proto := string(p.Proto())
			if len(bindings) == 0 {
				detail.Ports = append(detail.Ports, Port{PrivatePort: uint16(priv), Type: proto})
				continue
			}
			for _, b := range bindings {
				pub, _ := strconv.ParseUint(b.HostPort, 10, 16)
				ip := ""
				if b.HostIP.IsValid() {
					ip = b.HostIP.String()
				}
				detail.Ports = append(detail.Ports, Port{
					PrivatePort: uint16(priv),
					PublicPort:  uint16(pub),
					Type:        proto,
					IP:          ip,
				})
			}
		}
		for name, ep := range c.NetworkSettings.Networks {
			attach := NetworkAttachment{Name: name}
			if ep != nil && ep.IPAddress.IsValid() {
				attach.IP = ep.IPAddress.String()
			}
			detail.Networks = append(detail.Networks, attach)
		}
	}
	for _, mnt := range c.Mounts {
		detail.Mounts = append(detail.Mounts, Mount{
			Type:        string(mnt.Type),
			Source:      mnt.Source,
			Destination: mnt.Destination,
			Name:        mnt.Name,
			RW:          mnt.RW,
		})
	}
	return detail, nil
}

type Stats struct {
	CPUPercent float64
	MemUsage   uint64
	MemLimit   uint64
	MemPercent float64
	NetRx      uint64
	NetTx      uint64
	BlockRead  uint64
	BlockWrite uint64
	Pids       uint64
}

func (m *Manager) ContainerStats(ctx context.Context, id int64, host, containerID string) (<-chan Stats, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	res, err := cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{Stream: true})
	if err != nil {
		return nil, err
	}

	out := make(chan Stats)
	go func() {
		defer close(out)
		defer func() { _ = res.Body.Close() }()
		decoder := json.NewDecoder(res.Body)
		for {
			var raw container.StatsResponse
			if derr := decoder.Decode(&raw); derr != nil {
				return
			}
			select {
			case out <- computeStats(raw):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func computeStats(s container.StatsResponse) Stats {
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemUsage) - float64(s.PreCPUStats.SystemUsage)
	onlineCPUs := float64(s.CPUStats.OnlineCPUs)
	if onlineCPUs == 0 {
		onlineCPUs = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
	}
	cpuPercent := 0.0
	if sysDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / sysDelta) * onlineCPUs * 100
	}

	memUsed := s.MemoryStats.Usage
	if inactive, ok := s.MemoryStats.Stats["inactive_file"]; ok && inactive <= memUsed {
		memUsed -= inactive
	} else if cache, ok := s.MemoryStats.Stats["cache"]; ok && cache <= memUsed {
		memUsed -= cache
	}
	memPercent := 0.0
	if s.MemoryStats.Limit > 0 {
		memPercent = float64(memUsed) / float64(s.MemoryStats.Limit) * 100
	}

	var netRx, netTx uint64
	for _, n := range s.Networks {
		netRx += n.RxBytes
		netTx += n.TxBytes
	}
	var blockRead, blockWrite uint64
	for _, b := range s.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(b.Op) {
		case "read":
			blockRead += b.Value
		case "write":
			blockWrite += b.Value
		}
	}

	return Stats{
		CPUPercent: cpuPercent,
		MemUsage:   memUsed,
		MemLimit:   s.MemoryStats.Limit,
		MemPercent: memPercent,
		NetRx:      netRx,
		NetTx:      netTx,
		BlockRead:  blockRead,
		BlockWrite: blockWrite,
		Pids:       s.PidsStats.Current,
	}
}

type Image struct {
	ID      string
	Tags    []string
	Size    int64
	Created int64
	InUse   bool
}

func (m *Manager) Images(ctx context.Context, id int64, host string) ([]Image, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	used := make(map[string]bool)
	if containers, cerr := cli.ContainerList(ctx, client.ContainerListOptions{All: true}); cerr == nil {
		for _, c := range containers.Items {
			used[c.ImageID] = true
		}
	}

	out := make([]Image, 0, len(res.Items))
	for _, img := range res.Items {
		tags := make([]string, 0, len(img.RepoTags))
		for _, tag := range img.RepoTags {
			if tag == "<none>:<none>" {
				continue
			}
			tags = append(tags, tag)
		}
		out = append(out, Image{
			ID:      strings.TrimPrefix(img.ID, "sha256:"),
			Tags:    tags,
			Size:    img.Size,
			Created: img.Created,
			InUse:   used[img.ID],
		})
	}
	return out, nil
}

func (m *Manager) ImageAction(ctx context.Context, id int64, host, imageID, action string) error {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	switch action {
	case "remove":
		_, err = cli.ImageRemove(ctx, imageID, client.ImageRemoveOptions{Force: true, PruneChildren: true})
	default:
		return fmt.Errorf("unknown image action %q", action)
	}
	return err
}

type PullProgress struct {
	Status  string
	ID      string
	Current int64
	Total   int64
	Error   string
}

func (m *Manager) ImagePull(ctx context.Context, id int64, host, ref string) (<-chan PullProgress, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	resp, err := cli.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return nil, err
	}

	out := make(chan PullProgress)
	go func() {
		defer close(out)
		defer func() { _ = resp.Close() }()
		for msg, merr := range resp.JSONMessages(ctx) {
			var p PullProgress
			if merr != nil {
				p.Error = merr.Error()
			} else {
				p.Status = msg.Status
				p.ID = msg.ID
				if msg.Progress != nil {
					p.Current = msg.Progress.Current
					p.Total = msg.Progress.Total
				}
				if msg.Error != nil {
					p.Error = msg.Error.Message
				}
			}
			select {
			case out <- p:
			case <-ctx.Done():
				return
			}
			if merr != nil {
				return
			}
		}
	}()
	return out, nil
}

type PruneResult struct {
	ImagesDeleted  int
	SpaceReclaimed uint64
}

func (m *Manager) ImagesPrune(ctx context.Context, id int64, host string, all bool) (PruneResult, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return PruneResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	imageIDs := make(map[string]bool)
	if list, lerr := cli.ImageList(ctx, client.ImageListOptions{}); lerr == nil {
		for _, img := range list.Items {
			imageIDs[img.ID] = true
		}
	}

	filters := client.Filters{}
	if all {
		filters = filters.Add("dangling", "false")
	}
	res, err := cli.ImagePrune(ctx, client.ImagePruneOptions{Filters: filters})
	if err != nil {
		return PruneResult{}, err
	}

	removed := 0
	for _, d := range res.Report.ImagesDeleted {
		if d.Deleted != "" && imageIDs[d.Deleted] {
			removed++
		}
	}
	return PruneResult{
		ImagesDeleted:  removed,
		SpaceReclaimed: res.Report.SpaceReclaimed,
	}, nil
}

type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Stack      string
	Created    int64
	InUse      bool
}

func (m *Manager) Volumes(ctx context.Context, id int64, host string) ([]Volume, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return nil, err
	}

	used := make(map[string]bool)
	if containers, cerr := cli.ContainerList(ctx, client.ContainerListOptions{All: true}); cerr == nil {
		for _, c := range containers.Items {
			for _, mnt := range c.Mounts {
				if mnt.Name != "" {
					used[mnt.Name] = true
				}
			}
		}
	}

	out := make([]Volume, 0, len(res.Items))
	for _, v := range res.Items {
		created := int64(0)
		if t, perr := time.Parse(time.RFC3339, v.CreatedAt); perr == nil {
			created = t.Unix()
		}
		out = append(out, Volume{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Stack:      v.Labels[composeProjectLabel],
			Created:    created,
			InUse:      used[v.Name],
		})
	}
	return out, nil
}

func (m *Manager) VolumeAction(ctx context.Context, id int64, host, volumeName, action string) error {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	switch action {
	case "remove":
		_, err = cli.VolumeRemove(ctx, volumeName, client.VolumeRemoveOptions{Force: true})
	default:
		return fmt.Errorf("unknown volume action %q", action)
	}
	return err
}

type VolumeCreateInput struct {
	Name   string
	Driver string
}

func (m *Manager) VolumeCreate(ctx context.Context, id int64, host string, in VolumeCreateInput) (Volume, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return Volume{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	driver := in.Driver
	if driver == "" {
		driver = "local"
	}
	res, err := cli.VolumeCreate(ctx, client.VolumeCreateOptions{
		Name:   in.Name,
		Driver: driver,
	})
	if err != nil {
		return Volume{}, err
	}

	v := res.Volume
	created := int64(0)
	if t, perr := time.Parse(time.RFC3339, v.CreatedAt); perr == nil {
		created = t.Unix()
	}
	return Volume{
		Name:       v.Name,
		Driver:     v.Driver,
		Mountpoint: v.Mountpoint,
		Stack:      v.Labels[composeProjectLabel],
		Created:    created,
		InUse:      false,
	}, nil
}

var predefinedNetworks = map[string]bool{"bridge": true, "host": true, "none": true}

type Network struct {
	ID      string
	Name    string
	Driver  string
	Scope   string
	Stack   string
	Created int64
	InUse   bool
}

func (m *Manager) Networks(ctx context.Context, id int64, host string) ([]Network, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	used := make(map[string]bool)
	if containers, cerr := cli.ContainerList(ctx, client.ContainerListOptions{All: true}); cerr == nil {
		for _, c := range containers.Items {
			if c.NetworkSettings == nil {
				continue
			}
			for name := range c.NetworkSettings.Networks {
				used[name] = true
			}
		}
	}

	out := make([]Network, 0, len(res.Items))
	for _, n := range res.Items {
		out = append(out, Network{
			ID:      n.ID,
			Name:    n.Name,
			Driver:  n.Driver,
			Scope:   n.Scope,
			Stack:   n.Labels[composeProjectLabel],
			Created: n.Created.Unix(),
			InUse:   used[n.Name] || predefinedNetworks[n.Name],
		})
	}
	return out, nil
}

func (m *Manager) NetworkAction(ctx context.Context, id int64, host, networkID, action string) error {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	switch action {
	case "remove":
		_, err = cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})
	default:
		return fmt.Errorf("unknown network action %q", action)
	}
	return err
}

type NetworkCreateInput struct {
	Name   string
	Driver string
	Subnet string
}

type CreatedNetwork struct {
	ID      string
	Warning string
}

func (m *Manager) NetworkCreate(ctx context.Context, id int64, host string, in NetworkCreateInput) (CreatedNetwork, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return CreatedNetwork{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	driver := in.Driver
	if driver == "" {
		driver = "bridge"
	}
	opts := client.NetworkCreateOptions{Driver: driver}
	if in.Subnet != "" {
		prefix, perr := netip.ParsePrefix(in.Subnet)
		if perr != nil {
			return CreatedNetwork{}, fmt.Errorf("invalid subnet: %w", perr)
		}
		opts.IPAM = &network.IPAM{Config: []network.IPAMConfig{{Subnet: prefix}}}
	}

	res, err := cli.NetworkCreate(ctx, in.Name, opts)
	if err != nil {
		return CreatedNetwork{}, err
	}
	return CreatedNetwork{ID: res.ID, Warning: strings.Join(res.Warning, "; ")}, nil
}

type Stack struct {
	Name       string
	Type       string
	Services   int
	Running    int
	Total      int
	State      string
	WorkingDir string
}

func (m *Manager) Stacks(ctx context.Context, id int64, host string) ([]Stack, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	type group struct {
		stack    *Stack
		services map[string]bool
	}
	groups := make(map[string]*group)
	order := make([]string, 0)
	for _, c := range res.Items {
		project := c.Labels[composeProjectLabel]
		if project == "" {
			continue
		}
		g := groups[project]
		if g == nil {
			g = &group{
				stack:    &Stack{Name: project, Type: "external", WorkingDir: c.Labels[composeWorkingDirLabel]},
				services: make(map[string]bool),
			}
			groups[project] = g
			order = append(order, project)
		}
		g.stack.Total++
		if c.State == "running" {
			g.stack.Running++
		}
		if service := c.Labels[composeServiceLabel]; service != "" {
			g.services[service] = true
		}
	}

	out := make([]Stack, 0, len(order))
	for _, project := range order {
		g := groups[project]
		g.stack.Services = len(g.services)
		switch g.stack.Running {
		case 0:
			g.stack.State = "stopped"
		case g.stack.Total:
			g.stack.State = "running"
		default:
			g.stack.State = "partial"
		}
		out = append(out, *g.stack)
	}
	return out, nil
}

func (m *Manager) StackAction(ctx context.Context, id int64, host, project, action string) error {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return err
	}
	listCtx, cancel := context.WithTimeout(ctx, callTimeout)
	res, err := cli.ContainerList(listCtx, client.ContainerListOptions{All: true})
	cancel()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, c := range res.Items {
		if c.Labels[composeProjectLabel] != project {
			continue
		}
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			actionCtx, actionCancel := context.WithTimeout(ctx, actionTimeout)
			defer actionCancel()
			if aerr := applyContainerAction(actionCtx, cli, containerID, action); aerr != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = aerr
				}
				mu.Unlock()
			}
		}(c.ID)
	}
	wg.Wait()
	return firstErr
}

type LogLine struct {
	Stream  string
	Message string
}

func (m *Manager) ContainerLogs(ctx context.Context, id int64, host, containerID string, tail int, follow bool) (<-chan LogLine, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}

	inspectCtx, cancel := context.WithTimeout(ctx, callTimeout)
	info, err := cli.ContainerInspect(inspectCtx, containerID, client.ContainerInspectOptions{})
	cancel()
	if err != nil {
		return nil, err
	}
	tty := info.Container.Config != nil && info.Container.Config.Tty

	tailValue := "all"
	if tail > 0 {
		tailValue = strconv.Itoa(tail)
	}
	res, err := cli.ContainerLogs(ctx, containerID, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tailValue,
	})
	if err != nil {
		return nil, err
	}

	out := make(chan LogLine)
	go func() {
		defer close(out)
		defer func() { _ = res.Close() }()
		if tty {
			w := &logWriter{ctx: ctx, out: out, stream: "stdout"}
			_, _ = io.Copy(w, res)
			w.flush()
			return
		}
		stdout := &logWriter{ctx: ctx, out: out, stream: "stdout"}
		stderr := &logWriter{ctx: ctx, out: out, stream: "stderr"}
		_, _ = stdcopy.StdCopy(stdout, stderr, res)
		stdout.flush()
		stderr.flush()
	}()
	return out, nil
}

type logWriter struct {
	ctx    context.Context
	out    chan<- LogLine
	stream string
	buf    []byte
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := bytes.TrimRight(w.buf[:i], "\r")
		w.buf = w.buf[i+1:]
		if !w.emit(string(line)) {
			return 0, context.Canceled
		}
	}
	return len(p), nil
}

func (w *logWriter) flush() {
	if len(w.buf) == 0 {
		return
	}
	w.emit(string(bytes.TrimRight(w.buf, "\r")))
	w.buf = nil
}

func (w *logWriter) emit(message string) bool {
	select {
	case <-w.ctx.Done():
		return false
	case w.out <- LogLine{Stream: w.stream, Message: message}:
		return true
	}
}

type ExecSession struct {
	cli    *client.Client
	execID string
	resp   client.HijackedResponse
	closed sync.Once
}

func (m *Manager) ContainerExec(ctx context.Context, id int64, host, containerID string) (*ExecSession, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}

	created, err := cli.ExecCreate(ctx, containerID, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Cmd:          execShell,
	})
	if err != nil {
		return nil, err
	}

	attached, err := cli.ExecAttach(ctx, created.ID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		return nil, err
	}

	return &ExecSession{cli: cli, execID: created.ID, resp: attached.HijackedResponse}, nil
}

func (s *ExecSession) Stdin() io.Writer {
	return s.resp.Conn
}

func (s *ExecSession) Stdout() io.Reader {
	return s.resp.Reader
}

func (s *ExecSession) Resize(ctx context.Context, rows, cols uint) error {
	_, err := s.cli.ExecResize(ctx, s.execID, client.ExecResizeOptions{Height: rows, Width: cols})
	return err
}

func (s *ExecSession) Close() {
	s.closed.Do(func() { s.resp.Close() })
}

func (m *Manager) ContainerAction(ctx context.Context, id int64, host, containerID, action string) error {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	return applyContainerAction(ctx, cli, containerID, action)
}

func applyContainerAction(ctx context.Context, cli *client.Client, containerID, action string) error {
	var err error
	switch action {
	case "start":
		_, err = cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	case "stop":
		_, err = cli.ContainerStop(ctx, containerID, client.ContainerStopOptions{})
	case "restart":
		_, err = cli.ContainerRestart(ctx, containerID, client.ContainerRestartOptions{})
	case "pause":
		_, err = cli.ContainerPause(ctx, containerID, client.ContainerPauseOptions{})
	case "unpause":
		_, err = cli.ContainerUnpause(ctx, containerID, client.ContainerUnpauseOptions{})
	case "kill":
		_, err = cli.ContainerKill(ctx, containerID, client.ContainerKillOptions{})
	case "remove":
		_, err = cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{Force: true})
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	return err
}

func (m *Manager) WatchEvents(ctx context.Context, id int64, host string) (<-chan struct{}, <-chan error) {
	out := make(chan struct{})
	errc := make(chan error, 1)

	cli, err := m.clientFor(id, host)
	if err != nil {
		errc <- err
		close(out)
		return out, errc
	}

	go func() {
		defer close(out)
		res := cli.Events(ctx, client.EventsListOptions{})
		for {
			select {
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			case eerr := <-res.Err:
				errc <- eerr
				return
			case msg := <-res.Messages:
				if !eventIsMeaningful(msg) {
					continue
				}
				select {
				case out <- struct{}{}:
				case <-ctx.Done():
					errc <- ctx.Err()
					return
				}
			}
		}
	}()

	return out, errc
}

func eventIsMeaningful(msg events.Message) bool {
	switch msg.Type {
	case events.ContainerEventType:
		action := string(msg.Action)
		if strings.HasPrefix(action, "exec_") || strings.HasPrefix(action, "health_status") {
			return false
		}
		switch action {
		case "top", "resize", "attach", "detach":
			return false
		}
		return true
	case events.ImageEventType, events.VolumeEventType, events.NetworkEventType:
		return true
	default:
		return false
	}
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.clients {
		_ = c.Close()
		delete(m.clients, id)
	}
}

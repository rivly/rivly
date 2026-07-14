package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

const actionTimeout = 60 * time.Second

var execShell = []string{"/bin/sh", "-c", "exec $(command -v bash || command -v sh)"}

const callTimeout = 5 * time.Second

const composeProjectLabel = "com.docker.compose.project"

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

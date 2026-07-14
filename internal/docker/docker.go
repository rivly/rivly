package docker

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

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

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.clients {
		_ = c.Close()
		delete(m.clients, id)
	}
}

package docker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/moby/moby/client"
)

const actionTimeout = 60 * time.Second

const pullCreateTimeout = 10 * time.Minute

const callTimeout = 5 * time.Second

const (
	composeProjectLabel    = "com.docker.compose.project"
	composeServiceLabel    = "com.docker.compose.service"
	composeWorkingDirLabel = "com.docker.compose.project.working_dir"
)

var ErrImagePull = errors.New("pull image")

type AuthResolver func(ctx context.Context, ref string) string

type cachedClient struct {
	host string
	cli  *client.Client
}

type Manager struct {
	mu      sync.Mutex
	clients map[int64]cachedClient
	authFor AuthResolver
}

func NewManager(authFor AuthResolver) *Manager {
	return &Manager{clients: make(map[int64]cachedClient), authFor: authFor}
}

type Client struct {
	cli     *client.Client
	authFor AuthResolver
}

func (m *Manager) Client(id int64, host string) (*Client, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli, authFor: m.authFor}, nil
}

func (d *Client) registryAuth(ctx context.Context, ref string) string {
	if d.authFor == nil {
		return ""
	}
	return d.authFor(ctx, ref)
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
		if c.host == host {
			return c.cli, nil
		}
		_ = c.cli.Close()
		delete(m.clients, id)
	}
	cli, err := client.New(client.WithHost(host))
	if err != nil {
		return nil, err
	}
	m.clients[id] = cachedClient{host: host, cli: cli}
	return cli, nil
}

func (d *Client) Info(ctx context.Context) (SystemInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := d.cli.Info(ctx, client.InfoOptions{})
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

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.clients {
		_ = c.cli.Close()
		delete(m.clients, id)
	}
}

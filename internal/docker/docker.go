package docker

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/client"
)

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

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, c := range m.clients {
		_ = c.Close()
		delete(m.clients, id)
	}
}

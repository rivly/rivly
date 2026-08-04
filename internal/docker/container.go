package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

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

type PortMapping struct {
	HostPort      string
	ContainerPort string
	Proto         string
}

type MountInput struct {
	Source   string
	Target   string
	ReadOnly bool
}

type ContainerCreateInput struct {
	Name          string
	Image         string
	Command       []string
	Env           []string
	Ports         []PortMapping
	Mounts        []MountInput
	Network       string
	RestartPolicy string
	Start         bool
}

func (m *Manager) ContainerCreate(ctx context.Context, id int64, host string, in ContainerCreateInput) (string, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, pullCreateTimeout)
	defer cancel()

	config := &container.Config{
		Image: in.Image,
		Env:   in.Env,
		Cmd:   in.Command,
	}
	hostConfig := &container.HostConfig{}

	if len(in.Ports) > 0 {
		exposed := network.PortSet{}
		bindings := network.PortMap{}
		for _, p := range in.Ports {
			proto := p.Proto
			if proto == "" {
				proto = "tcp"
			}
			port, perr := network.ParsePort(p.ContainerPort + "/" + proto)
			if perr != nil {
				return "", fmt.Errorf("invalid container port %q: %w", p.ContainerPort, perr)
			}
			exposed[port] = struct{}{}
			if p.HostPort != "" {
				bindings[port] = append(bindings[port], network.PortBinding{HostPort: p.HostPort})
			}
		}
		config.ExposedPorts = exposed
		hostConfig.PortBindings = bindings
	}

	for _, mnt := range in.Mounts {
		mountType := mount.TypeVolume
		if strings.HasPrefix(mnt.Source, "/") || strings.HasPrefix(mnt.Source, ".") {
			mountType = mount.TypeBind
		}
		hostConfig.Mounts = append(hostConfig.Mounts, mount.Mount{
			Type:     mountType,
			Source:   mnt.Source,
			Target:   mnt.Target,
			ReadOnly: mnt.ReadOnly,
		})
	}

	if in.RestartPolicy != "" && in.RestartPolicy != "no" {
		hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(in.RestartPolicy)}
	}
	if in.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(in.Network)
	}

	opts := client.ContainerCreateOptions{
		Name:       in.Name,
		Config:     config,
		HostConfig: hostConfig,
	}
	res, err := cli.ContainerCreate(ctx, opts)
	if err != nil && strings.Contains(err.Error(), "No such image") {
		if perr := m.pullImage(ctx, cli, in.Image); perr != nil {
			return "", fmt.Errorf("pull image %q: %w", in.Image, perr)
		}
		res, err = cli.ContainerCreate(ctx, opts)
	}
	if err != nil {
		return "", err
	}

	if in.Start {
		if _, err := cli.ContainerStart(ctx, res.ID, client.ContainerStartOptions{}); err != nil {
			return res.ID, fmt.Errorf("container created but failed to start: %w", err)
		}
	}
	return res.ID, nil
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

package docker

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

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

type NetworkSubnet struct {
	Subnet  string
	Gateway string
}

type NetworkContainer struct {
	ID   string
	Name string
	IPv4 string
}

type NetworkDetail struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Internal   bool
	Attachable bool
	Created    int64
	Subnets    []NetworkSubnet
	Labels     map[string]string
	Containers []NetworkContainer
}

func (m *Manager) NetworkDetail(ctx context.Context, id int64, host, networkID string) (NetworkDetail, error) {
	cli, err := m.clientFor(id, host)
	if err != nil {
		return NetworkDetail{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	res, err := cli.NetworkInspect(ctx, networkID, client.NetworkInspectOptions{})
	if err != nil {
		return NetworkDetail{}, err
	}
	n := res.Network
	detail := NetworkDetail{
		ID:         n.ID,
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Internal:   n.Internal,
		Attachable: n.Attachable,
		Created:    n.Created.Unix(),
		Labels:     n.Labels,
	}
	for _, cfg := range n.IPAM.Config {
		subnet := ""
		if cfg.Subnet.IsValid() {
			subnet = cfg.Subnet.String()
		}
		gateway := ""
		if cfg.Gateway.IsValid() {
			gateway = cfg.Gateway.String()
		}
		if subnet != "" || gateway != "" {
			detail.Subnets = append(detail.Subnets, NetworkSubnet{Subnet: subnet, Gateway: gateway})
		}
	}
	for cid, ep := range n.Containers {
		ipv4 := ""
		if ep.IPv4Address.IsValid() {
			ipv4 = ep.IPv4Address.String()
		}
		detail.Containers = append(detail.Containers, NetworkContainer{ID: cid, Name: ep.Name, IPv4: ipv4})
	}
	return detail, nil
}

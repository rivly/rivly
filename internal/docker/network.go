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

func (d *Client) Networks(ctx context.Context) ([]Network, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := d.cli.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	used := make(map[string]bool)
	if containers, cerr := d.cli.ContainerList(ctx, client.ContainerListOptions{All: true}); cerr == nil {
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

func (d *Client) NetworkAction(ctx context.Context, networkID, action string) error {
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	var err error
	switch action {
	case "remove":
		_, err = d.cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})
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

func (d *Client) NetworkCreate(ctx context.Context, in NetworkCreateInput) (CreatedNetwork, error) {
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

	res, err := d.cli.NetworkCreate(ctx, in.Name, opts)
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

func (d *Client) NetworkDetail(ctx context.Context, networkID string) (NetworkDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	res, err := d.cli.NetworkInspect(ctx, networkID, client.NetworkInspectOptions{})
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

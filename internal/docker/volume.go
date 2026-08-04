package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/moby/moby/client"
)

type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Stack      string
	Created    int64
	InUse      bool
}

func (d *Client) Volumes(ctx context.Context) ([]Volume, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := d.cli.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return nil, err
	}

	used := make(map[string]bool)
	if containers, cerr := d.cli.ContainerList(ctx, client.ContainerListOptions{All: true}); cerr == nil {
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

func (d *Client) VolumeAction(ctx context.Context, volumeName, action string) error {
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	var err error
	switch action {
	case "remove":
		_, err = d.cli.VolumeRemove(ctx, volumeName, client.VolumeRemoveOptions{Force: true})
	default:
		return fmt.Errorf("unknown volume action %q", action)
	}
	return err
}

type VolumeCreateInput struct {
	Name   string
	Driver string
}

func (d *Client) VolumeCreate(ctx context.Context, in VolumeCreateInput) (Volume, error) {
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	driver := in.Driver
	if driver == "" {
		driver = "local"
	}
	res, err := d.cli.VolumeCreate(ctx, client.VolumeCreateOptions{
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

type VolumeContainer struct {
	ID   string
	Name string
}

type VolumeDetail struct {
	Name       string
	Driver     string
	Mountpoint string
	Scope      string
	Created    int64
	Labels     map[string]string
	Options    map[string]string
	Containers []VolumeContainer
}

func (d *Client) VolumeDetail(ctx context.Context, name string) (VolumeDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	res, err := d.cli.VolumeInspect(ctx, name, client.VolumeInspectOptions{})
	if err != nil {
		return VolumeDetail{}, err
	}
	v := res.Volume
	created := int64(0)
	if t, perr := time.Parse(time.RFC3339, v.CreatedAt); perr == nil {
		created = t.Unix()
	}
	detail := VolumeDetail{
		Name:       v.Name,
		Driver:     v.Driver,
		Mountpoint: v.Mountpoint,
		Scope:      v.Scope,
		Created:    created,
		Labels:     v.Labels,
		Options:    v.Options,
	}

	if containers, cerr := d.cli.ContainerList(ctx, client.ContainerListOptions{All: true}); cerr == nil {
		for _, c := range containers.Items {
			for _, mnt := range c.Mounts {
				if mnt.Name != name {
					continue
				}
				cname := ""
				if len(c.Names) > 0 {
					cname = strings.TrimPrefix(c.Names[0], "/")
				}
				detail.Containers = append(detail.Containers, VolumeContainer{ID: c.ID, Name: cname})
				break
			}
		}
	}
	return detail, nil
}

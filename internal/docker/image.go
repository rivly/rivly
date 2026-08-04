package docker

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/moby/moby/client"
)

type Image struct {
	ID      string
	Tags    []string
	Size    int64
	Created int64
	InUse   bool
}

func (d *Client) Images(ctx context.Context) ([]Image, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	res, err := d.cli.ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	used := make(map[string]bool)
	if containers, cerr := d.cli.ContainerList(ctx, client.ContainerListOptions{All: true}); cerr == nil {
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

func (d *Client) ImageAction(ctx context.Context, imageID, action string) error {
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	var err error
	switch action {
	case "remove":
		_, err = d.cli.ImageRemove(ctx, imageID, client.ImageRemoveOptions{Force: true, PruneChildren: true})
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

func (d *Client) ImagePull(ctx context.Context, ref string) (<-chan PullProgress, error) {
	resp, err := d.cli.ImagePull(ctx, ref, client.ImagePullOptions{RegistryAuth: d.registryAuth(ctx, ref)})
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

func (d *Client) ImagesPrune(ctx context.Context, all bool) (PruneResult, error) {
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	imageIDs := make(map[string]bool)
	if list, lerr := d.cli.ImageList(ctx, client.ImageListOptions{}); lerr == nil {
		for _, img := range list.Items {
			imageIDs[img.ID] = true
		}
	}

	filters := client.Filters{}
	if all {
		filters = filters.Add("dangling", "false")
	}
	res, err := d.cli.ImagePrune(ctx, client.ImagePruneOptions{Filters: filters})
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

type ImageContainer struct {
	ID   string
	Name string
}

type ImageDetail struct {
	ID           string
	Tags         []string
	Digests      []string
	Size         int64
	Created      int64
	Architecture string
	Os           string
	Author       string
	WorkingDir   string
	Command      []string
	Entrypoint   []string
	Env          []string
	ExposedPorts []string
	Labels       map[string]string
	Containers   []ImageContainer
}

func (d *Client) ImageDetail(ctx context.Context, imageID string) (ImageDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	res, err := d.cli.ImageInspect(ctx, imageID)
	if err != nil {
		return ImageDetail{}, err
	}
	r := res.InspectResponse

	created := int64(0)
	if t, perr := time.Parse(time.RFC3339, r.Created); perr == nil {
		created = t.Unix()
	}
	detail := ImageDetail{
		ID:           strings.TrimPrefix(r.ID, "sha256:"),
		Tags:         r.RepoTags,
		Digests:      r.RepoDigests,
		Size:         r.Size,
		Created:      created,
		Architecture: r.Architecture,
		Os:           r.Os,
		Author:       r.Author,
	}
	if r.Config != nil {
		detail.WorkingDir = r.Config.WorkingDir
		detail.Command = r.Config.Cmd
		detail.Entrypoint = r.Config.Entrypoint
		detail.Env = r.Config.Env
		detail.Labels = r.Config.Labels
		for port := range r.Config.ExposedPorts {
			detail.ExposedPorts = append(detail.ExposedPorts, port)
		}
		sort.Strings(detail.ExposedPorts)
	}

	if containers, cerr := d.cli.ContainerList(ctx, client.ContainerListOptions{All: true}); cerr == nil {
		for _, c := range containers.Items {
			if c.ImageID != r.ID {
				continue
			}
			cname := ""
			if len(c.Names) > 0 {
				cname = strings.TrimPrefix(c.Names[0], "/")
			}
			detail.Containers = append(detail.Containers, ImageContainer{ID: c.ID, Name: cname})
		}
	}
	return detail, nil
}

func (d *Client) pullImage(ctx context.Context, ref string) error {
	resp, err := d.cli.ImagePull(ctx, ref, client.ImagePullOptions{RegistryAuth: d.registryAuth(ctx, ref)})
	if err != nil {
		return err
	}
	defer func() { _ = resp.Close() }()
	for msg, merr := range resp.JSONMessages(ctx) {
		if merr != nil {
			return merr
		}
		if msg.Error != nil {
			return fmt.Errorf("%s", msg.Error.Message)
		}
	}
	return ctx.Err()
}

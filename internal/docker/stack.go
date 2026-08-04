package docker

import (
	"context"
	"sync"

	"github.com/moby/moby/client"
)

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

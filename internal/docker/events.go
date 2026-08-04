package docker

import (
	"context"
	"strings"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

func (d *Client) WatchEvents(ctx context.Context) (<-chan struct{}, <-chan error) {
	out := make(chan struct{})
	errc := make(chan error, 1)

	go func() {
		defer close(out)
		res := d.cli.Events(ctx, client.EventsListOptions{})
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

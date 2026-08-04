package environment

import (
	"context"
	"time"

	"github.com/rivly/rivly/internal/database/db"
)

const (
	eventDebounce       = 250 * time.Millisecond
	watcherReconnect    = 3 * time.Second
	watcherConnectGrace = 2 * time.Second
)

var watcherRefresh = 30 * time.Second

func (s *Service) RunWatchers(ctx context.Context) {
	watched := make(map[int64]bool)
	for {
		s.startWatchers(ctx, watched)
		select {
		case <-ctx.Done():
			return
		case <-time.After(watcherRefresh):
		}
	}
}

func (s *Service) startWatchers(ctx context.Context, watched map[int64]bool) {
	envs, err := s.queries.ListEnvironments(ctx)
	if err != nil {
		s.logger.Error("watcher: could not list environments", "err", err)
		return
	}
	for _, e := range envs {
		if watched[e.ID] {
			continue
		}
		watched[e.ID] = true
		s.spawn(func() { s.watch(ctx, e) })
	}
}

func (s *Service) watch(ctx context.Context, e db.Environment) {
	first := true
	for ctx.Err() == nil {
		signals, errc := s.docker.WatchEvents(ctx, e.ID, e.Url)
		s.consume(ctx, e, signals, errc, first)
		first = false
		select {
		case <-ctx.Done():
			return
		case <-time.After(watcherReconnect):
		}
	}
}

func (s *Service) consume(
	ctx context.Context,
	e db.Environment,
	signals <-chan struct{},
	errc <-chan error,
	first bool,
) {
	debounce := time.NewTimer(eventDebounce)
	debounce.Stop()
	defer debounce.Stop()

	var connected <-chan time.Time
	if !first {
		grace := time.NewTimer(watcherConnectGrace)
		defer grace.Stop()
		connected = grace.C
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-errc:
			return
		case <-connected:
			s.Publish(ctx, e)
			connected = nil
		case _, ok := <-signals:
			if !ok {
				return
			}
			debounce.Reset(eventDebounce)
		case <-debounce.C:
			s.Publish(ctx, e)
		}
	}
}

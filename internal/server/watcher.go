package server

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

func (s *Server) RunWatchers(ctx context.Context) {
	envs, err := s.queries.ListEnvironments(ctx)
	if err != nil {
		s.logger.Error("watcher: could not list environments", "err", err)
		return
	}
	for _, e := range envs {
		go s.watchEnvironment(ctx, e)
	}
}

func (s *Server) watchEnvironment(ctx context.Context, e db.Environment) {
	first := true
	for ctx.Err() == nil {
		signals, errc := s.docker.WatchEvents(ctx, e.ID, e.Url)
		s.consumeEvents(ctx, e, signals, errc, first)
		first = false
		select {
		case <-ctx.Done():
			return
		case <-time.After(watcherReconnect):
		}
	}
}

func (s *Server) consumeEvents(
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
			s.publishEnvironment(ctx, e)
			connected = nil
		case _, ok := <-signals:
			if !ok {
				return
			}
			debounce.Reset(eventDebounce)
		case <-debounce.C:
			s.publishEnvironment(ctx, e)
		}
	}
}

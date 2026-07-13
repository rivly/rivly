package server

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/rivly/rivly/internal/database/db"
)

func (s *Server) RunPoller(ctx context.Context) {
	interval := s.cfg.PollInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.pollEnvironments(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.pollEnvironments(ctx)
		}
	}
}

func (s *Server) pollEnvironments(ctx context.Context) {
	envs, err := s.queries.ListEnvironments(ctx)
	if err != nil {
		s.logger.Error("poller: could not list environments", "err", err)
		return
	}

	details := make([]environmentDetailResponse, len(envs))
	var wg sync.WaitGroup
	for i, e := range envs {
		wg.Add(1)
		go func(i int, e db.Environment) {
			defer wg.Done()
			details[i] = s.buildEnvironment(ctx, e)
		}(i, e)
	}
	wg.Wait()

	for _, detail := range details {
		fingerprint := detail
		fingerprint.LastSeen = nil
		key, err := json.Marshal(fingerprint)
		if err != nil {
			continue
		}
		if s.lastEnvState[detail.ID] == string(key) {
			continue
		}
		s.lastEnvState[detail.ID] = string(key)
		s.events.Publish("environment.updated", detail)
	}
}

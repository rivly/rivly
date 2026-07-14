package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		key := s.envFingerprint(ctx, detail)
		if key == "" {
			continue
		}
		s.envStateMu.Lock()
		changed := s.lastEnvState[detail.ID] != key
		if changed {
			s.lastEnvState[detail.ID] = key
		}
		s.envStateMu.Unlock()
		if changed {
			s.events.Publish("environment.updated", detail)
		}
	}
}

func (s *Server) publishEnvironment(ctx context.Context, e db.Environment) {
	detail := s.buildEnvironment(ctx, e)
	if key := s.envFingerprint(ctx, detail); key != "" {
		s.envStateMu.Lock()
		s.lastEnvState[detail.ID] = key
		s.envStateMu.Unlock()
	}
	s.events.Publish("environment.updated", detail)
}

func (s *Server) envFingerprint(ctx context.Context, detail environmentDetailResponse) string {
	fingerprint := detail
	fingerprint.LastSeen = nil
	key, err := json.Marshal(fingerprint)
	if err != nil {
		return ""
	}
	return string(key) + s.stacksSignature(ctx, detail.ID)
}

func (s *Server) stacksSignature(ctx context.Context, envID int64) string {
	stacks, err := s.queries.ListStacks(ctx, envID)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, st := range stacks {
		fmt.Fprintf(&b, "|%s:%d", st.Name, st.UpdatedAt)
	}
	return b.String()
}

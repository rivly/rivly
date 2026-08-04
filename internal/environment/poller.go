package environment

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rivly/rivly/internal/database/db"
)

func (s *Service) RunPoller(ctx context.Context) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	s.poll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *Service) poll(ctx context.Context) {
	details, err := s.BuildAll(ctx)
	if err != nil {
		s.logger.Error("poller: could not list environments", "err", err)
		return
	}

	for _, detail := range details {
		if s.changed(ctx, detail) {
			s.observe(ctx, detail)
		}
	}
}

func (s *Service) Publish(ctx context.Context, e db.Environment) {
	detail := s.Build(ctx, e)
	s.changed(ctx, detail)
	s.observe(ctx, detail)
}

func (s *Service) changed(ctx context.Context, detail Detail) bool {
	key := s.fingerprint(ctx, detail)
	if key == "" {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastState[detail.ID] == key {
		return false
	}
	s.lastState[detail.ID] = key
	return true
}

func (s *Service) fingerprint(ctx context.Context, detail Detail) string {
	stable := detail
	stable.LastSeen = nil

	key, err := json.Marshal(stable)
	if err != nil {
		return ""
	}
	if s.sign == nil {
		return string(key)
	}
	return string(key) + s.sign(ctx, detail.ID)
}

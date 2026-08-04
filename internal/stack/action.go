package stack

import (
	"context"
	"sync"

	"github.com/rivly/rivly/internal/database/db"
)

var validActions = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"remove":  true,
}

type ActionResult struct {
	Name  string
	OK    bool
	Error string
}

func ValidAction(action string) bool {
	return validActions[action]
}

func (s *Service) Act(ctx context.Context, env db.Environment, action string, names []string) []ActionResult {
	managed := s.managedByName(ctx, env.ID)

	results := make([]ActionResult, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = s.act(ctx, env, name, action, managed)
		}()
	}
	wg.Wait()
	return results
}

func (s *Service) act(ctx context.Context, env db.Environment, name, action string, managed map[string]db.Stack) ActionResult {
	if action == "remove" {
		if record, ok := managed[name]; ok {
			return s.remove(ctx, env, record)
		}
	}

	client, err := s.docker(env.ID, env.Url)
	if err != nil {
		s.logger.Warn("stack action failed", "action", action, "stack", name, "err", err)
		return ActionResult{Name: name, OK: false, Error: "action failed"}
	}
	if err := client.StackAction(ctx, name, action); err != nil {
		s.logger.Warn("stack action failed", "action", action, "stack", name, "err", err)
		return ActionResult{Name: name, OK: false, Error: "action failed"}
	}
	return ActionResult{Name: name, OK: true}
}

func (s *Service) remove(ctx context.Context, env db.Environment, record db.Stack) ActionResult {
	envContent := EnvFileContent(ParseEnvVars(record.Env))

	var out string
	var err error
	if record.Source == SourceGit {
		if !s.Acquire(record.ID) {
			return ActionResult{Name: record.Name, OK: false, Error: "an update is running"}
		}
		defer s.Release(record.ID)
		out, err = s.compose.RemoveRepo(ctx, env.Url, env.ID, record.Name, record.GitPath, envContent)
	} else {
		out, err = s.compose.Remove(ctx, env.Url, env.ID, record.Name, record.Content, envContent)
	}
	if err != nil {
		s.logger.Warn("managed stack remove failed", "stack", record.Name, "err", err, "out", out)
		return ActionResult{Name: record.Name, OK: false, Error: "action failed"}
	}

	if err := s.queries.DeleteStack(ctx, db.DeleteStackParams{EnvID: env.ID, Name: record.Name}); err != nil {
		s.logger.Error("could not delete stack record", "stack", record.Name, "err", err)
	}
	return ActionResult{Name: record.Name, OK: true}
}

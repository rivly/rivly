package stack

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/rivly/rivly/internal/database/db"
)

type Summary struct {
	Name       string
	Type       string
	Source     string
	Services   int
	Running    int
	Total      int
	State      string
	WorkingDir string
	CreatedAt  int64
	UpdatedAt  int64
	CreatedBy  string
	UpdatedBy  string
}

type GitDetail struct {
	URL           string
	Ref           string
	Path          string
	CredentialID  int64
	Commit        string
	AutoUpdate    bool
	PollInterval  int64
	LastCheckedAt int64
	LastError     string
}

type Detail struct {
	Name    string
	Source  string
	Content string
	Env     []EnvVar
	Git     *GitDetail
}

func (s *Service) List(ctx context.Context, env db.Environment) ([]Summary, error) {
	discovered, err := s.docker.Stacks(ctx, env.ID, env.Url)
	if err != nil {
		return nil, unreachable("environment is unreachable")
	}

	managed := s.managedByName(ctx, env.ID)

	merged := make(map[string]Summary, len(discovered))
	for _, d := range discovered {
		sum := Summary{
			Name:       d.Name,
			Type:       d.Type,
			Services:   d.Services,
			Running:    d.Running,
			Total:      d.Total,
			State:      d.State,
			WorkingDir: d.WorkingDir,
		}
		if m, ok := managed[d.Name]; ok {
			sum.Type = "rivly"
			sum.Source = m.Source
			sum.CreatedAt = m.CreatedAt
			sum.UpdatedAt = m.UpdatedAt
			sum.CreatedBy = m.CreatedBy
			sum.UpdatedBy = m.UpdatedBy
		}
		merged[d.Name] = sum
	}
	for name, m := range managed {
		if _, ok := merged[name]; ok {
			continue
		}
		merged[name] = Summary{
			Name:      name,
			Type:      "rivly",
			Source:    m.Source,
			State:     "stopped",
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			CreatedBy: m.CreatedBy,
			UpdatedBy: m.UpdatedBy,
		}
	}

	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]Summary, 0, len(names))
	for _, name := range names {
		out = append(out, merged[name])
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, env db.Environment, name string) (Detail, error) {
	record, err := s.queries.GetStack(ctx, db.GetStackParams{EnvID: env.ID, Name: name})
	if errors.Is(err, sql.ErrNoRows) {
		return Detail{}, notFound("stack not found")
	}
	if err != nil {
		return Detail{}, fmt.Errorf("load stack %q: %w", name, err)
	}

	detail := Detail{
		Name:    record.Name,
		Source:  record.Source,
		Content: record.Content,
		Env:     ParseEnvVars(record.Env),
	}
	if record.Source == SourceGit {
		detail.Git = &GitDetail{
			URL:           record.GitUrl,
			Ref:           record.GitRef,
			Path:          record.GitPath,
			CredentialID:  record.GitCredentialID,
			Commit:        record.GitCommit,
			AutoUpdate:    record.GitAutoUpdate == 1,
			PollInterval:  record.GitPollInterval,
			LastCheckedAt: record.GitLastCheckedAt,
			LastError:     record.GitLastError,
		}
	}
	return detail, nil
}

func (s *Service) managedByName(ctx context.Context, envID int64) map[string]db.Stack {
	managed := make(map[string]db.Stack)
	list, err := s.queries.ListStacks(ctx, envID)
	if err != nil {
		s.logger.Error("could not list managed stacks", "env", envID, "err", err)
		return managed
	}
	for _, m := range list {
		managed[m.Name] = m
	}
	return managed
}

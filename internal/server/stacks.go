package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/database/db"
)

type stackResponse struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Services   int    `json:"services"`
	Running    int    `json:"running"`
	Total      int    `json:"total"`
	State      string `json:"state"`
	WorkingDir string `json:"workingDir"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
	CreatedBy  string `json:"createdBy"`
	UpdatedBy  string `json:"updatedBy"`
}

var validStackActions = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"remove":  true,
}

var stackNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

func (s *Server) handleListStacks(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	discovered, err := s.docker.Stacks(r.Context(), env.ID, env.Url)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "environment is unreachable")
		return
	}

	managed := make(map[string]db.Stack)
	if list, lerr := s.queries.ListStacks(r.Context(), env.ID); lerr == nil {
		for _, m := range list {
			managed[m.Name] = m
		}
	}

	merged := make(map[string]stackResponse, len(discovered))
	for _, d := range discovered {
		sr := stackResponse{
			Name:       d.Name,
			Type:       d.Type,
			Services:   d.Services,
			Running:    d.Running,
			Total:      d.Total,
			State:      d.State,
			WorkingDir: d.WorkingDir,
		}
		if m, ok := managed[d.Name]; ok {
			sr.Type = "rivly"
			sr.CreatedAt = m.CreatedAt
			sr.UpdatedAt = m.UpdatedAt
			sr.CreatedBy = m.CreatedBy
			sr.UpdatedBy = m.UpdatedBy
		}
		merged[d.Name] = sr
	}
	for name, m := range managed {
		if _, ok := merged[name]; !ok {
			merged[name] = stackResponse{
				Name:      name,
				Type:      "rivly",
				State:     "stopped",
				CreatedAt: m.CreatedAt,
				UpdatedAt: m.UpdatedAt,
				CreatedBy: m.CreatedBy,
				UpdatedBy: m.UpdatedBy,
			}
		}
	}

	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]stackResponse, 0, len(names))
	for _, name := range names {
		out = append(out, merged[name])
	}
	s.writeJSON(w, http.StatusOK, out)
}

type envVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func envFileContent(vars []envVar) string {
	var b strings.Builder
	for _, v := range vars {
		key := strings.TrimSpace(v.Key)
		if key == "" {
			continue
		}
		fmt.Fprintf(&b, "%s=%s\n", key, v.Value)
	}
	return b.String()
}

func parseEnvVars(stored string) []envVar {
	vars := []envVar{}
	if stored != "" {
		_ = json.Unmarshal([]byte(stored), &vars)
	}
	return vars
}

type deployStackRequest struct {
	Name    string   `json:"name"`
	Content string   `json:"content"`
	Env     []envVar `json:"env"`
}

func (s *Server) handleDeployStack(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req deployStackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if !stackNamePattern.MatchString(name) {
		s.writeError(w, http.StatusBadRequest, "name must be lowercase letters, digits, - or _")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		s.writeError(w, http.StatusBadRequest, "compose file is empty")
		return
	}

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	_, getErr := s.queries.GetStack(r.Context(), db.GetStackParams{EnvID: env.ID, Name: name})
	isNew := errors.Is(getErr, sql.ErrNoRows)

	envJSON, err := json.Marshal(req.Env)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment variables")
		return
	}

	out, derr := s.compose.Deploy(r.Context(), env.Url, env.ID, name, req.Content, envFileContent(req.Env))
	if derr != nil {
		s.logger.Warn("stack deploy failed", "stack", name, "err", derr)
		if isNew {
			s.compose.Discard(r.Context(), env.Url, env.ID, name)
		}
		s.writeError(w, http.StatusUnprocessableEntity, composeError(out))
		return
	}

	author := ""
	if user, uerr := s.queries.GetUserByID(r.Context(), s.sessions.GetInt64(r.Context(), sessionUserID)); uerr == nil {
		author = user.DisplayName
	}

	if _, err := s.queries.UpsertStack(r.Context(), db.UpsertStackParams{
		EnvID:     env.ID,
		Name:      name,
		Content:   req.Content,
		Env:       string(envJSON),
		CreatedBy: author,
		UpdatedBy: author,
	}); err != nil {
		s.serverError(w, r, "could not save stack", err)
		return
	}

	s.publishEnvironment(r.Context(), env)
	s.writeJSON(w, http.StatusOK, map[string]string{"name": name})
}

func (s *Server) handleGetStack(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}
	name := chi.URLParam(r, "name")

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	stack, err := s.queries.GetStack(r.Context(), db.GetStackParams{EnvID: env.ID, Name: name})
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "stack not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load stack", err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"name":    stack.Name,
		"content": stack.Content,
		"env":     parseEnvVars(stack.Env),
	})
}

func (s *Server) handleStackActions(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid environment id")
		return
	}

	var req bulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validStackActions[req.Action] {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid stack selection")
		return
	}

	env, err := s.queries.GetEnvironment(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeError(w, http.StatusNotFound, "environment not found")
		return
	}
	if err != nil {
		s.serverError(w, r, "could not load environment", err)
		return
	}

	managed := make(map[string]db.Stack)
	if list, lerr := s.queries.ListStacks(r.Context(), env.ID); lerr == nil {
		for _, m := range list {
			managed[m.Name] = m
		}
	}

	results := make([]actionResult, len(req.IDs))
	var wg sync.WaitGroup
	for i, project := range req.IDs {
		wg.Add(1)
		go func(i int, project string) {
			defer wg.Done()
			results[i] = s.runStackAction(r.Context(), env, project, req.Action, managed)
		}(i, project)
	}
	wg.Wait()

	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) runStackAction(ctx context.Context, env db.Environment, project, action string, managed map[string]db.Stack) actionResult {
	if action == "remove" {
		if stack, ok := managed[project]; ok {
			out, derr := s.compose.Remove(ctx, env.Url, env.ID, project, stack.Content, envFileContent(parseEnvVars(stack.Env)))
			if derr != nil {
				s.logger.Warn("managed stack remove failed", "stack", project, "err", derr, "out", out)
				return actionResult{ID: project, OK: false, Error: "action failed"}
			}
			if derr := s.queries.DeleteStack(ctx, db.DeleteStackParams{EnvID: env.ID, Name: project}); derr != nil {
				s.logger.Error("could not delete stack record", "stack", project, "err", derr)
			}
			return actionResult{ID: project, OK: true}
		}
	}

	if err := s.docker.StackAction(ctx, env.ID, env.Url, project, action); err != nil {
		s.logger.Warn("stack action failed", "action", action, "stack", project, "err", err)
		return actionResult{ID: project, OK: false, Error: "action failed"}
	}
	return actionResult{ID: project, OK: true}
}

func composeError(out string) string {
	if out == "" {
		return "deployment failed"
	}
	if len(out) > 4000 {
		out = out[len(out)-4000:]
	}
	return out
}

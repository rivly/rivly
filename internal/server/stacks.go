package server

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rivly/rivly/internal/stack"
)

type stackResponse struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Source     string `json:"source"`
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

type gitDetail struct {
	URL           string `json:"url"`
	Ref           string `json:"ref"`
	Path          string `json:"path"`
	CredentialID  int64  `json:"credentialId"`
	Commit        string `json:"commit"`
	AutoUpdate    bool   `json:"autoUpdate"`
	PollInterval  int64  `json:"pollInterval"`
	LastCheckedAt int64  `json:"lastCheckedAt"`
	LastError     string `json:"lastError"`
}

type stackDetailResponse struct {
	Name    string         `json:"name"`
	Source  string         `json:"source"`
	Content string         `json:"content"`
	Env     []stack.EnvVar `json:"env"`
	Git     *gitDetail     `json:"git"`
}

type gitSource struct {
	URL          string `json:"url"`
	Ref          string `json:"ref"`
	Path         string `json:"path"`
	CredentialID int64  `json:"credentialId"`
	AutoUpdate   bool   `json:"autoUpdate"`
	PollInterval int64  `json:"pollInterval"`
}

type deployStackRequest struct {
	Name    string         `json:"name"`
	Source  string         `json:"source"`
	Content string         `json:"content"`
	Env     []stack.EnvVar `json:"env"`
	Git     *gitSource     `json:"git"`
}

var stackStatus = map[stack.Kind]int{
	stack.KindInvalid:     http.StatusBadRequest,
	stack.KindNotFound:    http.StatusNotFound,
	stack.KindConflict:    http.StatusConflict,
	stack.KindRejected:    http.StatusUnprocessableEntity,
	stack.KindUnreachable: http.StatusBadGateway,
}

func (s *Server) stackError(w http.ResponseWriter, r *http.Request, message string, err error) {
	var known *stack.Error
	if errors.As(err, &known) {
		s.writeError(w, stackStatus[known.Kind], known.Message)
		return
	}
	s.serverError(w, r, message, err)
}

func (s *Server) handleListStacks(w http.ResponseWriter, r *http.Request) {
	summaries, err := s.stacks.List(r.Context(), environmentFrom(r))
	if err != nil {
		s.stackError(w, r, "could not list stacks", err)
		return
	}

	out := make([]stackResponse, 0, len(summaries))
	for _, sum := range summaries {
		out = append(out, stackResponse(sum))
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetStack(w http.ResponseWriter, r *http.Request) {
	detail, err := s.stacks.Get(r.Context(), environmentFrom(r), chi.URLParam(r, "name"))
	if err != nil {
		s.stackError(w, r, "could not load stack", err)
		return
	}

	out := stackDetailResponse{
		Name:    detail.Name,
		Source:  detail.Source,
		Content: detail.Content,
		Env:     detail.Env,
	}
	if detail.Git != nil {
		out.Git = &gitDetail{
			URL:           detail.Git.URL,
			Ref:           detail.Git.Ref,
			Path:          detail.Git.Path,
			CredentialID:  detail.Git.CredentialID,
			Commit:        detail.Git.Commit,
			AutoUpdate:    detail.Git.AutoUpdate,
			PollInterval:  detail.Git.PollInterval,
			LastCheckedAt: detail.Git.LastCheckedAt,
			LastError:     detail.Git.LastError,
		}
	}
	s.writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDeployStack(w http.ResponseWriter, r *http.Request) {
	var req deployStackRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}

	in := stack.DeployInput{
		Name:    req.Name,
		Source:  req.Source,
		Content: req.Content,
		Env:     req.Env,
		Author:  s.currentUserName(r),
	}
	if req.Git != nil {
		in.Git = &stack.GitSource{
			URL:          req.Git.URL,
			Ref:          req.Git.Ref,
			Path:         req.Git.Path,
			CredentialID: req.Git.CredentialID,
			AutoUpdate:   req.Git.AutoUpdate,
			PollInterval: req.Git.PollInterval,
		}
	}

	env := environmentFrom(r)
	if err := s.stacks.Deploy(r.Context(), env, in); err != nil {
		s.stackError(w, r, "could not deploy the stack", err)
		return
	}

	s.publishEnvironment(r.Context(), env)
	s.writeJSON(w, http.StatusOK, map[string]string{"name": req.Name})
}

func (s *Server) handleStackActions(w http.ResponseWriter, r *http.Request) {
	var req bulkActionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.badRequest(w, err)
		return
	}
	if !stack.ValidAction(req.Action) {
		s.writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > maxBulkActions {
		s.writeError(w, http.StatusBadRequest, "invalid stack selection")
		return
	}

	done := s.stacks.Act(r.Context(), environmentFrom(r), req.Action, req.IDs)

	results := make([]actionResult, 0, len(done))
	for _, d := range done {
		results = append(results, actionResult{ID: d.Name, OK: d.OK, Error: d.Error})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

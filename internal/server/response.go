package server

import (
	"encoding/json"
	"errors"
	"net/http"
)

const maxRequestBody = 1 << 20

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("encode response", "err", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) serverError(w http.ResponseWriter, r *http.Request, message string, err error) {
	s.logger.Error(message, "err", err, "method", r.Method, "path", r.URL.Path)
	s.writeError(w, http.StatusInternalServerError, message)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (s *Server) badRequest(w http.ResponseWriter, err error) {
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		s.writeError(w, http.StatusRequestEntityTooLarge, "request body is too large")
		return
	}
	s.writeError(w, http.StatusBadRequest, "invalid request body")
}

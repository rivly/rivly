package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const (
	indexFile      = "index.html"
	assetsPrefix   = "/assets/"
	immutableCache = "public, max-age=31536000, immutable"
)

func (s *Server) dashboardHandler(assets fs.FS) http.HandlerFunc {
	files := http.FileServerFS(assets)

	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			s.writeError(w, http.StatusNotFound, "not found")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		name := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if name == "." || name == "/" {
			name = indexFile
		}

		if _, err := fs.Stat(assets, name); err != nil {
			s.serveIndex(w, r, assets)
			return
		}

		if strings.HasPrefix(r.URL.Path, assetsPrefix) {
			w.Header().Set("Cache-Control", immutableCache)
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	}
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request, assets fs.FS) {
	index, err := fs.ReadFile(assets, indexFile)
	if err != nil {
		s.serverError(w, r, "could not read the dashboard", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(index); err != nil {
		s.logger.Debug("dashboard write failed", "err", err)
	}
}

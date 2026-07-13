package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

const sessionUserID = "userID"

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.sessions.GetInt64(r.Context(), sessionUserID) == 0 {
			s.writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		s.logger.LogAttrs(r.Context(), slog.LevelInfo, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", ww.Status()),
			slog.Int("bytes", ww.BytesWritten()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("ip", middleware.GetClientIP(r.Context())),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			s.logger.Error("panic recovered",
				"err", rec,
				"method", r.Method,
				"path", r.URL.Path,
				"stack", string(debug.Stack()),
			)
			s.writeError(w, http.StatusInternalServerError, "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func secureCookies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestIsHTTPS(r) {
			w = &secureCookieWriter{ResponseWriter: w}
		}
		next.ServeHTTP(w, r)
	})
}

type secureCookieWriter struct {
	http.ResponseWriter
	patched bool
}

func (w *secureCookieWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *secureCookieWriter) patch() {
	if w.patched {
		return
	}
	w.patched = true
	header := w.Header()
	cookies := header.Values("Set-Cookie")
	if len(cookies) == 0 {
		return
	}
	header.Del("Set-Cookie")
	for _, c := range cookies {
		if !strings.Contains(c, "; Secure") {
			c += "; Secure"
		}
		header.Add("Set-Cookie", c)
	}
}

func (w *secureCookieWriter) WriteHeader(status int) {
	w.patch()
	w.ResponseWriter.WriteHeader(status)
}

func (w *secureCookieWriter) Write(b []byte) (int, error) {
	w.patch()
	return w.ResponseWriter.Write(b)
}

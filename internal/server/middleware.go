package server

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rivly/rivly/internal/database/db"
)

const sessionUserID = "userID"

type contextKey int

const (
	environmentKey contextKey = iota
	dockerKey
)

func (s *Server) withEnvironment(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		client, err := s.docker(env.ID, env.Url)
		if err != nil {
			s.serverError(w, r, "could not reach the environment", err)
			return
		}

		ctx := context.WithValue(r.Context(), environmentKey, env)
		ctx = context.WithValue(ctx, dockerKey, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func dockerFrom(r *http.Request) DockerClient {
	client, _ := r.Context().Value(dockerKey).(DockerClient)
	return client
}

func environmentFrom(r *http.Request) db.Environment {
	env, _ := r.Context().Value(environmentKey).(db.Environment)
	return env
}

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
			if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
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

var contentSecurityPolicy = strings.Join([]string{
	"default-src 'self'",
	"script-src 'self'",
	"style-src 'self' 'unsafe-inline'",
	"img-src 'self' data:",
	"font-src 'self'",
	"connect-src 'self'",
	"object-src 'none'",
	"base-uri 'none'",
	"form-action 'self'",
	"frame-ancestors 'none'",
}, "; ")

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Content-Security-Policy", contentSecurityPolicy)
		header.Set("Referrer-Policy", "no-referrer")
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

func (w *secureCookieWriter) Flush() {
	w.patch()
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *secureCookieWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

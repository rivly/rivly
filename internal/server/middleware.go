package server

import (
	"net/http"
	"strings"
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

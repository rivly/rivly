package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

func NewSessionManager(db *sql.DB) *scs.SessionManager {
	sm := scs.New()
	sm.Store = sqlite3store.New(db)
	sm.Lifetime = 7 * 24 * time.Hour
	sm.IdleTimeout = 3 * 24 * time.Hour
	sm.Cookie.Name = "rivly_session"
	sm.Cookie.Path = "/"
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = false
	return sm
}

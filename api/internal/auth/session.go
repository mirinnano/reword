package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/mirinnano/reword/api/internal/platform/httpx"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

const (
	SessionTTL = 30 * 24 * time.Hour
	// Sliding expiry is refreshed at most this often to avoid a write per request.
	sessionTouchEvery = time.Hour
)

// Session is the signed-in state attached to a request context.
type Session struct {
	TokenHash       []byte
	UserID          uint64
	Email           string
	Role            store.UsersRole
	ActiveProfileID uint64 // 0 when none selected
}

type sessionKey struct{}

func FromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(*Session)
	return s, ok
}

type Sessions struct {
	Q      *store.Queries
	Secure bool
	Now    func() time.Time
	Log    *slog.Logger
}

// CookieName uses the __Host- prefix whenever the cookie is Secure, which
// pins it to this exact origin (no Domain attribute, Path=/).
func (m *Sessions) CookieName() string {
	if m.Secure {
		return "__Host-reword_session"
	}
	return "reword_session"
}

// Start creates a session row and sets the cookie.
func (m *Sessions) Start(ctx context.Context, w http.ResponseWriter, r *http.Request, userID, profileID uint64) error {
	plain, hash, err := NewToken()
	if err != nil {
		return err
	}
	now := m.Now()
	ua := r.UserAgent()
	if len(ua) > 255 {
		ua = ua[:255]
	}
	if err := m.Q.CreateSession(ctx, store.CreateSessionParams{
		TokenHash:       hash,
		UserID:          userID,
		ActiveProfileID: nullID(profileID),
		Ip:              httpx.ClientIP(ctx),
		UserAgent:       ua,
		ExpiresAt:       now.Add(SessionTTL),
	}); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.CookieName(),
		Value:    plain,
		Path:     "/",
		MaxAge:   int(SessionTTL / time.Second),
		HttpOnly: true,
		Secure:   m.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (m *Sessions) Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: m.CookieName(), Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: m.Secure, SameSite: http.SameSiteLaxMode,
	})
}

// Middleware attaches the session (if any) to the request context.
func (m *Sessions) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(m.CookieName())
		if err != nil || c.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		now := m.Now()
		hash := HashToken(c.Value)
		row, err := m.Q.GetSession(r.Context(), store.GetSessionParams{TokenHash: hash, ExpiresAt: now})
		if errors.Is(err, sql.ErrNoRows) {
			m.Clear(w)
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			m.Log.Error("load session", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
			return
		}
		if now.Sub(row.LastSeenAt) > sessionTouchEvery {
			if err := m.Q.TouchSession(r.Context(), store.TouchSessionParams{
				LastSeenAt: now, ExpiresAt: now.Add(SessionTTL), TokenHash: hash,
			}); err != nil {
				m.Log.Warn("touch session", "err", err)
			}
		}
		s := &Session{TokenHash: hash, UserID: row.UserID, Email: row.Email, Role: row.Role}
		if row.ActiveProfileID.Valid {
			s.ActiveProfileID = uint64(row.ActiveProfileID.Int64)
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, s)))
	})
}

// Require rejects requests without a session.
func Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); !ok {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func nullID(id uint64) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(id), Valid: true}
}

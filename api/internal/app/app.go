// Package app wires feature packages into the HTTP handler and job worker.
package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mirinnano/reword/api/internal/account"
	"github.com/mirinnano/reword/api/internal/auth"
	"github.com/mirinnano/reword/api/internal/jobs"
	"github.com/mirinnano/reword/api/internal/mail"
	"github.com/mirinnano/reword/api/internal/platform/httpx"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

type Options struct {
	DB           *sql.DB
	Log          *slog.Logger
	AppOrigin    string
	CookieSecure bool
	Now          func() time.Time
	// Outbox defaults to the jobs table; tests substitute a recorder.
	Outbox mail.Outbox
	Sender mail.Sender
}

type App struct {
	Handler http.Handler
	Worker  *jobs.Worker
}

func New(o Options) *App {
	if o.Now == nil {
		o.Now = func() time.Time { return time.Now().UTC() }
	}
	q := store.New(o.DB)
	if o.Outbox == nil {
		o.Outbox = mail.JobOutbox{Q: q, Now: o.Now}
	}
	if o.Sender == nil {
		o.Sender = mail.LogSender{Log: o.Log}
	}

	sessions := &auth.Sessions{Q: q, Secure: o.CookieSecure, Now: o.Now, Log: o.Log}
	acct := &account.Handlers{Q: q, Log: o.Log}
	authH := &auth.Handlers{
		DB: o.DB, Q: q,
		Hasher:     auth.NewHasher(2),
		Sessions:   sessions,
		Limiter:    &auth.Limiter{Q: q, Now: o.Now},
		Outbox:     o.Outbox,
		AppOrigin:  o.AppOrigin,
		Log:        o.Log,
		Now:        o.Now,
		AfterLogin: acct.BuildMe,
	}

	handler := httpx.NewRouter(httpx.Deps{
		Log: o.Log,
		DB:  o.DB,
		Mount: func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth.SameOrigin(o.AppOrigin))
				r.Use(sessions.Middleware)
				acct.PublicRoutes(r)
				authH.Routes(r)
				r.Group(func(r chi.Router) {
					r.Use(auth.Require)
					acct.Routes(r)
				})
			})
		},
	})

	worker := &jobs.Worker{
		DB: o.DB, Q: q, Log: o.Log, Now: o.Now,
		PollEvery: 2 * time.Second, LockFor: 5 * time.Minute, SweepEvery: 10 * time.Minute,
		Handlers: map[string]jobs.Handler{
			mail.JobKind: func(ctx context.Context, payload []byte) error {
				var m mail.Message
				if err := json.Unmarshal(payload, &m); err != nil {
					return err
				}
				return o.Sender.Send(ctx, m)
			},
		},
		Sweep: func(ctx context.Context) { sweep(ctx, q, o.Log, o.Now()) },
	}
	return &App{Handler: handler, Worker: worker}
}

// sweep deletes expired rows so the small staging DB does not grow forever.
func sweep(ctx context.Context, q *store.Queries, log *slog.Logger, now time.Time) {
	steps := []struct {
		name string
		run  func() (int64, error)
	}{
		{"sessions", func() (int64, error) { return q.DeleteExpiredSessions(ctx, now) }},
		{"pending_signups", func() (int64, error) { return q.DeleteExpiredPendingSignups(ctx, now) }},
		{"rate_limits", func() (int64, error) { return q.DeleteOldRateLimits(ctx, now.Add(-2*time.Hour)) }},
		{"jobs", func() (int64, error) { return q.DeleteFinishedJobs(ctx, now.Add(-7*24*time.Hour)) }},
	}
	for _, s := range steps {
		if n, err := s.run(); err != nil {
			log.Warn("sweep", "table", s.name, "err", err)
		} else if n > 0 {
			log.Info("sweep", "table", s.name, "deleted", n)
		}
	}
}

// Package mail composes transactional emails and hands them to the job
// queue; a Sender delivers them from the worker.
package mail

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mirinnano/reword/api/internal/platform/store"
)

const JobKind = "mail.send"

type Message struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
}

// Outbox accepts messages for later delivery. Sending from a queue keeps
// response times independent of whether an email was sent, so they do not
// reveal which addresses are registered.
type Outbox interface {
	Enqueue(ctx context.Context, m Message) error
}

// JobOutbox stores messages as jobs.
type JobOutbox struct {
	Q   *store.Queries
	Now func() time.Time
}

func (o JobOutbox) Enqueue(ctx context.Context, m Message) error {
	payload, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = o.Q.EnqueueJob(ctx, store.EnqueueJobParams{Kind: JobKind, Payload: payload, RunAt: o.Now()})
	return err
}

// Sender delivers one message.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// LogSender writes messages to the log instead of sending them. It exists for
// local and staging only; config refuses it in production because the log
// would then hold live sign-in links.
type LogSender struct{ Log *slog.Logger }

func (s LogSender) Send(_ context.Context, m Message) error {
	s.Log.Info("mail (log driver)", "to", m.To, "subject", m.Subject, "text", m.Text)
	return nil
}

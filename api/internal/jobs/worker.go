// Package jobs runs queued background work from the jobs table.
package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mirinnano/reword/api/internal/platform/store"
)

type Handler func(ctx context.Context, payload []byte) error

type Worker struct {
	DB       *sql.DB
	Q        *store.Queries
	Log      *slog.Logger
	Now      func() time.Time
	Handlers map[string]Handler

	PollEvery  time.Duration
	LockFor    time.Duration
	SweepEvery time.Duration
	Sweep      func(ctx context.Context) // periodic housekeeping, optional
}

// Run processes jobs until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	poll := time.NewTicker(w.PollEvery)
	defer poll.Stop()
	sweep := time.NewTicker(w.SweepEvery)
	defer sweep.Stop()
	for {
		for w.runOne(ctx) {
		}
		select {
		case <-ctx.Done():
			return
		case <-sweep.C:
			if n, err := w.Q.RequeueStaleJobs(ctx, sql.NullTime{Time: w.Now(), Valid: true}); err != nil {
				w.Log.Warn("requeue stale jobs", "err", err)
			} else if n > 0 {
				w.Log.Warn("requeued stale jobs", "n", n)
			}
			if w.Sweep != nil {
				w.Sweep(ctx)
			}
		case <-poll.C:
		}
	}
}

// runOne claims and runs one job; it reports whether a job was found.
func (w *Worker) runOne(ctx context.Context) bool {
	job, ok, err := w.claim(ctx)
	if err != nil {
		if ctx.Err() == nil {
			w.Log.Error("claim job", "err", err)
		}
		return false
	}
	if !ok {
		return false
	}
	log := w.Log.With("job_id", job.ID, "kind", job.Kind, "attempt", job.Attempts+1)

	h, known := w.Handlers[job.Kind]
	if !known {
		err = fmt.Errorf("no handler for kind %q", job.Kind)
	} else {
		err = h(ctx, job.Payload)
	}
	if err == nil {
		if err := w.Q.MarkJobDone(ctx, job.ID); err != nil {
			log.Error("mark job done", "err", err)
		}
		return true
	}
	log.Warn("job failed", "err", err)
	backoff := time.Duration(1<<min(job.Attempts, 10)) * 10 * time.Second
	if err := w.Q.MarkJobFailed(ctx, store.MarkJobFailedParams{
		RunAt:     w.Now().Add(backoff),
		LastError: sql.NullString{String: err.Error(), Valid: true},
		ID:        job.ID,
	}); err != nil {
		log.Error("mark job failed", "err", err)
	}
	return true
}

func (w *Worker) claim(ctx context.Context) (store.SelectClaimableJobRow, bool, error) {
	tx, err := w.DB.BeginTx(ctx, nil)
	if err != nil {
		return store.SelectClaimableJobRow{}, false, err
	}
	defer tx.Rollback()
	q := w.Q.WithTx(tx)
	now := w.Now()
	job, err := q.SelectClaimableJob(ctx, now)
	if errors.Is(err, sql.ErrNoRows) {
		return job, false, nil
	}
	if err != nil {
		return job, false, err
	}
	if err := q.MarkJobRunning(ctx, store.MarkJobRunningParams{
		LockedUntil: sql.NullTime{Time: now.Add(w.LockFor), Valid: true},
		ID:          job.ID,
	}); err != nil {
		return job, false, err
	}
	return job, true, tx.Commit()
}

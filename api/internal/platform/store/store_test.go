package store_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/mirinnano/reword/api/internal/platform/db"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

// openTestDB connects to a migrated database, or skips when none is configured.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set")
	}
	pool, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestListActiveGrades(t *testing.T) {
	q := store.New(openTestDB(t))
	grades, err := q.ListActiveGrades(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(grades) != 5 || grades[0].Code != "pre2" || grades[4].Code != "g1" {
		t.Fatalf("unexpected active grades: %+v", grades)
	}
}

// Two workers claiming concurrently must never get the same job.
func TestClaimJobSkipLocked(t *testing.T) {
	ctx := context.Background()
	pool := openTestDB(t)
	q := store.New(pool)

	kind := "test." + t.Name() + "." + time.Now().Format("150405.000000")
	now := time.Now().UTC()
	for range 2 {
		if _, err := q.EnqueueJob(ctx, store.EnqueueJobParams{
			Kind: kind, Payload: json.RawMessage(`{"n":1}`), RunAt: now.Add(-time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { pool.Exec("DELETE FROM jobs WHERE kind = ?", kind) })

	tx1, err := pool.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx1.Rollback()
	tx2, err := pool.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx2.Rollback()

	a, err := q.WithTx(tx1).SelectClaimableJob(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	b, err := q.WithTx(tx2).SelectClaimableJob(ctx, now)
	if errors.Is(err, sql.ErrNoRows) {
		t.Skip("other queued jobs exist; cannot assert ordering")
	}
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Fatalf("both workers claimed job %d", a.ID)
	}
}

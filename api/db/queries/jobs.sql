-- name: ListActiveGrades :many
SELECT id, code, name_ja, sort_order
FROM grades
WHERE is_active = TRUE
ORDER BY sort_order;

-- name: EnqueueJob :execlastid
INSERT INTO jobs (kind, payload, run_at)
VALUES (?, ?, ?);

-- name: SelectClaimableJob :one
-- Must run inside a transaction; SKIP LOCKED lets concurrent workers
-- pass over rows another worker is already claiming.
SELECT id, kind, payload, attempts, max_attempts
FROM jobs
WHERE status = 'queued' AND run_at <= ?
ORDER BY run_at, id
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: MarkJobRunning :exec
UPDATE jobs
SET status = 'running', attempts = attempts + 1, locked_until = ?
WHERE id = ?;

-- name: MarkJobDone :exec
-- Payloads may hold one-time links, so they are cleared once delivered.
UPDATE jobs SET status = 'done', locked_until = NULL, payload = '{}' WHERE id = ?;

-- name: RequeueStaleJobs :execrows
-- Jobs left 'running' by a crashed worker go back to the queue.
UPDATE jobs SET status = 'queued', locked_until = NULL
WHERE status = 'running' AND locked_until < ?;

-- name: DeleteFinishedJobs :execrows
DELETE FROM jobs WHERE status = 'done' AND updated_at < ? LIMIT 1000;

-- name: MarkJobFailed :exec
-- Requeue with backoff until max_attempts, then park as failed.
UPDATE jobs
SET status = IF(attempts >= max_attempts, 'failed', 'queued'),
    run_at = ?,
    locked_until = NULL,
    last_error = ?
WHERE id = ?;

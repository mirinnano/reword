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
UPDATE jobs SET status = 'done', locked_until = NULL WHERE id = ?;

-- name: MarkJobFailed :exec
-- Requeue with backoff until max_attempts, then park as failed.
UPDATE jobs
SET status = IF(attempts >= max_attempts, 'failed', 'queued'),
    run_at = ?,
    locked_until = NULL,
    last_error = ?
WHERE id = ?;

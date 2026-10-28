-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, active_profile_id, ip, user_agent, expires_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetSession :one
SELECT s.token_hash, s.user_id, s.active_profile_id, s.last_seen_at, s.expires_at,
       u.email, u.role
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = ? AND s.expires_at > ?;

-- name: TouchSession :exec
UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE token_hash = ?;

-- name: SetSessionProfile :exec
UPDATE sessions SET active_profile_id = ? WHERE token_hash = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = ?;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions WHERE expires_at <= ? LIMIT 1000;

-- name: CreateEmailToken :exec
INSERT INTO email_tokens (token_hash, user_id, purpose, expires_at) VALUES (?, ?, ?, ?);

-- name: GetEmailTokenForUpdate :one
SELECT token_hash, user_id, purpose, expires_at, used_at
FROM email_tokens WHERE token_hash = ? FOR UPDATE;

-- name: MarkEmailTokenUsed :exec
UPDATE email_tokens SET used_at = ? WHERE token_hash = ?;

-- name: InvalidateEmailTokens :exec
-- Older unused tokens of the same purpose stop working once a new one is issued or one is used.
UPDATE email_tokens SET used_at = ?
WHERE user_id = ? AND purpose = ? AND used_at IS NULL;

-- name: HitRateLimit :exec
INSERT INTO rate_limits (bucket, window_start, hits) VALUES (?, ?, 1)
ON DUPLICATE KEY UPDATE hits = hits + 1;

-- name: GetRateLimitHits :one
SELECT hits FROM rate_limits WHERE bucket = ? AND window_start = ?;

-- name: DeleteOldRateLimits :execrows
DELETE FROM rate_limits WHERE window_start < ? LIMIT 1000;

-- name: CreatePendingSignup :exec
INSERT INTO pending_signups
  (token_hash, email, password_hash, display_name, target_grade_id, terms_version, privacy_version, ip, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPendingSignupForUpdate :one
SELECT token_hash, email, password_hash, display_name, target_grade_id,
       terms_version, privacy_version, ip, expires_at
FROM pending_signups WHERE token_hash = ? FOR UPDATE;

-- name: DeletePendingSignupsByEmail :exec
DELETE FROM pending_signups WHERE email = ?;

-- name: DeleteExpiredPendingSignups :execrows
DELETE FROM pending_signups WHERE expires_at <= ? LIMIT 1000;

-- name: CountPendingSignups :one
SELECT COUNT(*) FROM pending_signups WHERE email = ? AND expires_at > ?;

-- name: CreateUser :execlastid
INSERT INTO users (email, password_hash) VALUES (?, ?);

-- name: GetUserByEmail :one
SELECT id, email, password_hash, role, email_verified_at
FROM users WHERE email = ?;

-- name: GetUserByID :one
SELECT id, email, role, email_verified_at, created_at
FROM users WHERE id = ?;

-- name: MarkEmailVerified :exec
UPDATE users SET email_verified_at = ? WHERE id = ? AND email_verified_at IS NULL;

-- name: UpdatePasswordHash :exec
UPDATE users SET password_hash = ? WHERE id = ?;

-- name: CreateConsent :exec
INSERT INTO consents (user_id, document, version, ip) VALUES (?, ?, ?, ?);

-- name: CreateProfile :execlastid
INSERT INTO learner_profiles (user_id, display_name, target_grade_id) VALUES (?, ?, ?);

-- name: ListProfiles :many
SELECT id, display_name, target_grade_id, created_at
FROM learner_profiles WHERE user_id = ? ORDER BY id;

-- name: CountProfiles :one
SELECT COUNT(*) FROM learner_profiles WHERE user_id = ?;

-- name: GetProfile :one
SELECT id, user_id, display_name, target_grade_id, created_at
FROM learner_profiles WHERE id = ? AND user_id = ?;

-- name: UpdateProfile :execrows
UPDATE learner_profiles SET display_name = ?, target_grade_id = ?
WHERE id = ? AND user_id = ?;

-- name: DeleteProfile :execrows
DELETE FROM learner_profiles WHERE id = ? AND user_id = ?;

-- name: ListGrades :many
SELECT id, code, name_ja, sort_order, is_active FROM grades ORDER BY sort_order;

-- name: GradeIsActive :one
SELECT is_active FROM grades WHERE id = ?;

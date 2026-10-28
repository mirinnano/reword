-- name: ListPracticeTypes :many
SELECT t.id, t.code, t.name_ja, t.format, g.code AS grade_code, g.name_ja AS grade_name_ja,
       COUNT(q.id) AS question_count
FROM question_types t
JOIN grades g ON g.id = t.grade_id
LEFT JOIN questions q ON q.question_type_id = t.id AND q.status = 'published'
WHERE g.is_active = TRUE
GROUP BY t.id, t.code, t.name_ja, t.format, g.code, g.name_ja, g.sort_order, t.sort_order
HAVING question_count > 0
ORDER BY g.sort_order, t.sort_order;

-- name: GetQuestionTypeByCode :one
SELECT id, code, name_ja, format FROM question_types WHERE code = ?;

-- name: ListPublishedQuestionKeys :many
SELECT id, stable_key FROM questions
WHERE question_type_id = ? AND status = 'published';

-- name: ProfileHistoryByType :many
-- Per question key: how often this profile answered it, how often correctly,
-- and when last. Keys are compared across versions on purpose.
SELECT q.stable_key,
       CAST(COUNT(*) AS UNSIGNED) AS answered,
       CAST(SUM(ai.is_correct) AS UNSIGNED) AS correct,
       CAST(UNIX_TIMESTAMP(MAX(ai.answered_at)) AS UNSIGNED) AS last_answered_unix
FROM attempt_items ai
JOIN attempts a  ON a.id = ai.attempt_id
JOIN questions q ON q.id = ai.question_id
WHERE a.profile_id = ? AND q.question_type_id = ? AND ai.answered_at IS NOT NULL
GROUP BY q.stable_key;

-- name: CreateAttempt :execlastid
INSERT INTO attempts (profile_id, question_type_id, mode, item_count, started_at)
VALUES (?, ?, 'practice', ?, ?);

-- name: CreateAttemptItem :exec
INSERT INTO attempt_items (attempt_id, position, question_id, choice_order)
VALUES (?, ?, ?, ?);

-- name: GetAttempt :one
SELECT a.id, a.profile_id, a.item_count, a.answered_count, a.correct_count,
       a.started_at, a.finished_at, t.code AS type_code, t.name_ja AS type_name_ja,
       t.format AS type_format
FROM attempts a
JOIN question_types t ON t.id = a.question_type_id
WHERE a.id = ? AND a.profile_id = ?;

-- name: GetAttemptForUpdate :one
SELECT id, item_count, answered_count, correct_count, finished_at
FROM attempts WHERE id = ? AND profile_id = ? FOR UPDATE;

-- name: ListAttemptItems :many
SELECT ai.position, ai.choice_order, ai.selected_original, ai.is_correct, ai.answered_at,
       q.payload, q.answer, q.explanation_ja, q.vocab_keys, q.difficulty
FROM attempt_items ai
JOIN questions q ON q.id = ai.question_id
WHERE ai.attempt_id = ?
ORDER BY ai.position;

-- name: GetAttemptItemForUpdate :one
SELECT ai.id, ai.choice_order, ai.answered_at, q.payload, q.answer, q.explanation_ja, q.vocab_keys
FROM attempt_items ai
JOIN questions q ON q.id = ai.question_id
WHERE ai.attempt_id = ? AND ai.position = ?
FOR UPDATE;

-- name: AnswerAttemptItem :exec
UPDATE attempt_items
SET selected_original = ?, is_correct = ?, time_ms = ?, answered_at = ?
WHERE id = ?;

-- name: RecordAttemptAnswer :exec
UPDATE attempts
SET answered_count = answered_count + 1, correct_count = correct_count + ?
WHERE id = ?;

-- name: FinishAttempt :exec
UPDATE attempts SET finished_at = ? WHERE id = ? AND finished_at IS NULL;

-- name: ListRecentAttempts :many
SELECT a.id, t.code AS type_code, t.name_ja AS type_name_ja, a.item_count,
       a.answered_count, a.correct_count, a.started_at, a.finished_at
FROM attempts a
JOIN question_types t ON t.id = a.question_type_id
WHERE a.profile_id = ?
ORDER BY a.started_at DESC
LIMIT ?;

-- name: GetVocabByKeys :many
SELECT stable_key, headword, pos, payload FROM vocab_items
WHERE stable_key IN (sqlc.slice('keys')) AND status = 'published';

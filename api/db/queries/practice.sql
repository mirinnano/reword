-- name: ListPracticeTypes :many
SELECT t.id, t.code, t.name_ja, t.format, g.code AS grade_code, g.name_ja AS grade_name_ja,
       COUNT(q.id) AS question_count
FROM question_types t
JOIN grades g ON g.id = t.grade_id
LEFT JOIN questions q ON q.question_type_id = t.id AND q.status = 'published'
WHERE g.is_active = TRUE AND t.code <> 'g2-gr'
GROUP BY t.id, t.code, t.name_ja, t.format, g.code, g.name_ja, g.sort_order, t.sort_order
HAVING question_count > 0
ORDER BY g.sort_order, t.sort_order;

-- name: GetQuestionTypeByCode :one
SELECT id, code, name_ja, format FROM question_types WHERE code = ?;

-- name: ListPublishedQuestionKeys :many
SELECT id, stable_key FROM questions
WHERE question_type_id = ? AND status = 'published';

-- name: ProfileAnswersByType :many
-- Every answer this profile gave to questions of one type, oldest first per
-- key. Keys are compared across versions on purpose; the scheduler only
-- looks at the recent run of answers, so old versions fade out.
-- FLOOR because CAST rounds: an answer given at .5s or later would otherwise
-- land in the future, and a missed question would not be due straight away.
SELECT q.stable_key, ai.is_correct,
       CAST(FLOOR(UNIX_TIMESTAMP(ai.answered_at)) AS UNSIGNED) AS answered_unix
FROM attempt_items ai
JOIN attempts a  ON a.id = ai.attempt_id
JOIN questions q ON q.id = ai.question_id
WHERE a.profile_id = ? AND q.question_type_id = ? AND ai.answered_at IS NOT NULL
  AND a.mode <> 'placement'
ORDER BY q.stable_key, ai.answered_at;

-- name: ProfileAnswers :many
-- ProfileAnswersByType across every type, for the daily routine. Neither query
-- reads placement answers: a placement test must not fill the review queue.
SELECT q.stable_key, ai.is_correct,
       CAST(FLOOR(UNIX_TIMESTAMP(ai.answered_at)) AS UNSIGNED) AS answered_unix
FROM attempt_items ai
JOIN attempts a  ON a.id = ai.attempt_id
JOIN questions q ON q.id = ai.question_id
WHERE a.profile_id = ? AND ai.answered_at IS NOT NULL AND a.mode <> 'placement'
ORDER BY q.stable_key, ai.answered_at;

-- name: ListDailyCandidates :many
-- The daily routine asks each word once: the headword-to-meaning drill stands
-- for the word, so the reverse drill stays in format practice only.
SELECT q.id, q.stable_key, t.code AS type_code, t.format, q.payload, q.difficulty, g.code AS grade_code
FROM questions q
JOIN question_types t ON t.id = q.question_type_id
JOIN grades g ON g.id = t.grade_id
WHERE q.status = 'published' AND t.format <> 'vocab_word';

-- name: GetDailyAttempt :one
SELECT id FROM attempts WHERE profile_id = ? AND daily_date = ?;

-- name: CountDailyAttempts :one
SELECT COUNT(*) FROM attempts WHERE profile_id = ? AND mode = 'daily';

-- name: CreateDailyAttempt :execlastid
INSERT INTO attempts (profile_id, mode, daily_date, item_count, started_at)
VALUES (?, 'daily', ?, ?, ?);

-- name: CreateAttempt :execlastid
INSERT INTO attempts (profile_id, question_type_id, mode, item_count, started_at)
VALUES (?, ?, 'practice', ?, ?);

-- name: CreateAttemptItem :exec
INSERT INTO attempt_items (attempt_id, position, question_id, is_review, choice_order)
VALUES (?, ?, ?, ?, ?);

-- name: GetAttempt :one
SELECT a.id, a.profile_id, a.item_count, a.answered_count, a.correct_count,
       a.started_at, a.finished_at, t.code AS type_code, t.name_ja AS type_name_ja,
       t.format AS type_format, a.mode
FROM attempts a
LEFT JOIN question_types t ON t.id = a.question_type_id
WHERE a.id = ? AND a.profile_id = ?;

-- name: GetAttemptForUpdate :one
SELECT id, mode, item_count, answered_count, correct_count, finished_at
FROM attempts WHERE id = ? AND profile_id = ? FOR UPDATE;

-- name: ListAttemptItems :many
SELECT ai.position, ai.is_review, ai.choice_order, ai.selected_original, ai.is_correct, ai.answered_at,
       q.payload, q.answer, q.explanation_ja, q.vocab_keys, q.difficulty, t.format
FROM attempt_items ai
JOIN questions q ON q.id = ai.question_id
JOIN question_types t ON t.id = q.question_type_id
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
SELECT a.id, t.code AS type_code, t.name_ja AS type_name_ja, a.mode, a.item_count,
       a.answered_count, a.correct_count, a.started_at, a.finished_at
FROM attempts a
LEFT JOIN question_types t ON t.id = a.question_type_id
WHERE a.profile_id = ? AND a.mode <> 'placement'
ORDER BY a.started_at DESC
LIMIT ?;

-- name: GetVocabByKeys :many
SELECT stable_key, headword, pos, payload FROM vocab_items
WHERE stable_key IN (sqlc.slice('keys')) AND status = 'published';

-- name: ListPublishedQuestionPayloads :many
-- For passage types, whose session selection needs each question's passage.
SELECT id, stable_key, payload FROM questions
WHERE question_type_id = ? AND status = 'published';

-- name: ListPlacementCandidates :many
SELECT q.id, t.code AS type_code, q.difficulty
FROM questions q
JOIN question_types t ON t.id = q.question_type_id
WHERE q.status = 'published' AND t.code IN ('pre2-q1', 'g2-q1');

-- name: GetPlacementStage :one
SELECT placement_stage FROM learner_profiles WHERE id = ?;

-- name: SetPlacementStage :exec
UPDATE learner_profiles SET placement_stage = ? WHERE id = ?;

-- name: GetOpenPlacement :one
SELECT id FROM attempts
WHERE profile_id = ? AND mode = 'placement' AND finished_at IS NULL
ORDER BY started_at DESC LIMIT 1;

-- name: CreatePlacementAttempt :execlastid
INSERT INTO attempts (profile_id, mode, item_count, started_at)
VALUES (?, 'placement', ?, ?);

-- name: ListGrammarUnits :many
SELECT stable_key, group_ja, title, sort_order FROM grammar_units
WHERE status = 'published' ORDER BY sort_order;

-- name: GetGrammarUnit :one
SELECT stable_key, group_ja, title, sort_order, payload FROM grammar_units
WHERE stable_key = ? AND status = 'published';

-- name: ListGrammarResults :many
-- A learner's best finished sitting of each unit.
SELECT grammar_unit, CAST(MAX(correct_count) AS UNSIGNED) AS best
FROM attempts
WHERE profile_id = ? AND mode = 'grammar' AND finished_at IS NOT NULL
GROUP BY grammar_unit;

-- name: CreateGrammarAttempt :execlastid
INSERT INTO attempts (profile_id, question_type_id, mode, grammar_unit, item_count, started_at)
VALUES (?, ?, 'grammar', ?, ?, ?);

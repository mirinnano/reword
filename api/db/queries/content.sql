-- name: GetGradeIDByCode :one
SELECT id FROM grades WHERE code = ?;

-- name: UpsertQuestionType :exec
INSERT INTO question_types (code, grade_id, name_ja, format, sort_order)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE grade_id = VALUES(grade_id), name_ja = VALUES(name_ja),
  format = VALUES(format), sort_order = VALUES(sort_order);

-- name: GetQuestionTypeID :one
SELECT id FROM question_types WHERE code = ?;

-- name: ListQuestionVersionsByType :many
-- Every version of every question of one type, newest first per key.
SELECT id, stable_key, version, content_hash, status
FROM questions
WHERE question_type_id = ?
ORDER BY stable_key, version DESC;

-- name: InsertQuestion :exec
INSERT INTO questions
  (stable_key, version, question_type_id, payload, answer, explanation_ja,
   difficulty, vocab_keys, content_hash, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'published');

-- name: SetQuestionStatus :exec
UPDATE questions SET status = ? WHERE id = ?;

-- name: ListVocabByGrade :many
SELECT id, stable_key, content_hash, status FROM vocab_items WHERE grade_id = ?;

-- name: InsertVocab :exec
INSERT INTO vocab_items (stable_key, grade_id, headword, pos, payload, content_hash, status)
VALUES (?, ?, ?, ?, ?, ?, 'published');

-- name: UpdateVocab :exec
UPDATE vocab_items
SET headword = ?, pos = ?, payload = ?, content_hash = ?, status = 'published'
WHERE id = ?;

-- name: SetVocabStatus :exec
UPDATE vocab_items SET status = ? WHERE id = ?;

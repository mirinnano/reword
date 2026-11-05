DELETE FROM attempt_items WHERE question_id IN (
  SELECT q.id FROM questions q JOIN question_types t ON t.id = q.question_type_id
  WHERE t.format IN ('passage_blank','passage_question'));
DELETE FROM attempts WHERE question_type_id IN (
  SELECT id FROM question_types WHERE format IN ('passage_blank','passage_question'));
DELETE FROM questions WHERE question_type_id IN (
  SELECT id FROM question_types WHERE format IN ('passage_blank','passage_question'));
DELETE FROM question_types WHERE format IN ('passage_blank','passage_question');
ALTER TABLE question_types MODIFY format ENUM('mcq','vocab_meaning','vocab_word') NOT NULL;

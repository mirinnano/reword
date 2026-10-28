DELETE FROM attempt_items WHERE question_id IN (
  SELECT q.id FROM questions q JOIN question_types t ON t.id = q.question_type_id
  WHERE t.format <> 'mcq');
DELETE FROM attempts WHERE question_type_id IN (SELECT id FROM question_types WHERE format <> 'mcq');
DELETE FROM questions WHERE question_type_id IN (SELECT id FROM question_types WHERE format <> 'mcq');
DELETE FROM question_types WHERE format <> 'mcq';
ALTER TABLE question_types MODIFY format ENUM('mcq') NOT NULL;

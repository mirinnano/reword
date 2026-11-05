DELETE FROM attempts WHERE mode = 'daily';
ALTER TABLE attempt_items DROP COLUMN is_review;
ALTER TABLE attempts
  DROP KEY uq_attempts_daily,
  DROP COLUMN daily_date,
  MODIFY mode ENUM('practice') NOT NULL,
  MODIFY question_type_id SMALLINT UNSIGNED NOT NULL;

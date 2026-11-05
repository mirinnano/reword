-- The daily routine mixes question types, so an attempt no longer belongs to
-- one type. daily_date is the learner's study day; the unique key allows one
-- routine per profile per day and is what makes starting one idempotent.
ALTER TABLE attempts
  MODIFY question_type_id SMALLINT UNSIGNED NULL,
  MODIFY mode ENUM('practice','daily') NOT NULL,
  ADD COLUMN daily_date DATE NULL AFTER mode,
  ADD UNIQUE KEY uq_attempts_daily (profile_id, daily_date);

-- Whether the learner had seen the question before this session.
ALTER TABLE attempt_items
  ADD COLUMN is_review BOOLEAN NOT NULL DEFAULT FALSE AFTER question_id;

-- The placement test is an attempt too, so a sitting can be resumed like any
-- other. Its answers stay out of the review schedule (see ProfileAnswers).
-- A question answered with "don't know" has answered_at set, selected_original
-- NULL and is_correct FALSE.
ALTER TABLE attempts
  MODIFY mode ENUM('practice','daily','placement') NOT NULL;

-- Where the latest finished placement test put this learner: how many blocks
-- they passed. NULL means they have not taken one.
ALTER TABLE learner_profiles
  ADD COLUMN placement_stage TINYINT UNSIGNED NULL AFTER target_grade_id;

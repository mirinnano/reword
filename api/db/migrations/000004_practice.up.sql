-- Practice sessions (phase 3). One attempt is one sitting of N questions of
-- one type for one learner profile; mock tests will reuse it with a deadline.
CREATE TABLE attempts (
  id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  profile_id       BIGINT UNSIGNED NOT NULL,
  question_type_id SMALLINT UNSIGNED NOT NULL,
  mode             ENUM('practice') NOT NULL,
  item_count       SMALLINT UNSIGNED NOT NULL,
  answered_count   SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  correct_count    SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  started_at       DATETIME(6) NOT NULL,
  finished_at      DATETIME(6) NULL,
  KEY idx_attempts_profile (profile_id, started_at),
  CONSTRAINT fk_attempts_profile FOREIGN KEY (profile_id) REFERENCES learner_profiles (id) ON DELETE CASCADE,
  CONSTRAINT fk_attempts_type    FOREIGN KEY (question_type_id) REFERENCES question_types (id)
) ENGINE=InnoDB;

-- One row per presented question. question_id pins the exact version shown.
-- Choices are shuffled per presentation; choice_order maps displayed index
-- to the original index, so the stored answer is in original terms.
CREATE TABLE attempt_items (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  attempt_id        BIGINT UNSIGNED NOT NULL,
  position          SMALLINT UNSIGNED NOT NULL,
  question_id       BIGINT UNSIGNED NOT NULL,
  choice_order      JSON NOT NULL CHECK (JSON_VALID(choice_order)),
  selected_original TINYINT UNSIGNED NULL,
  is_correct        BOOLEAN NULL,
  time_ms           INT UNSIGNED NULL,
  answered_at       DATETIME(6) NULL,
  UNIQUE KEY uq_attempt_items_position (attempt_id, position),
  KEY idx_attempt_items_question (question_id),
  CONSTRAINT fk_attempt_items_attempt  FOREIGN KEY (attempt_id) REFERENCES attempts (id) ON DELETE CASCADE,
  CONSTRAINT fk_attempt_items_question FOREIGN KEY (question_id) REFERENCES questions (id)
) ENGINE=InnoDB;

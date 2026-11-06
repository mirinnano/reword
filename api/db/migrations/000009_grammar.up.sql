-- Grammar units are the lesson around a set of check questions. The questions
-- themselves are ordinary rows of questions; a unit's sitting is an attempt
-- that names the unit, which is also where completion is read from.
CREATE TABLE grammar_units (
  id           INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  stable_key   VARCHAR(64) NOT NULL,
  grade_id     TINYINT UNSIGNED NOT NULL,
  group_ja     VARCHAR(40) NOT NULL,
  title        VARCHAR(80) NOT NULL,
  sort_order   SMALLINT UNSIGNED NOT NULL,
  payload      JSON NOT NULL CHECK (JSON_VALID(payload)),
  content_hash BINARY(32) NOT NULL,
  status       ENUM('published','retired') NOT NULL,
  UNIQUE KEY uq_grammar_units_key (stable_key),
  CONSTRAINT fk_grammar_units_grade FOREIGN KEY (grade_id) REFERENCES grades (id)
) ENGINE=InnoDB;

ALTER TABLE attempts
  MODIFY mode ENUM('practice','daily','placement','grammar') NOT NULL,
  ADD COLUMN grammar_unit VARCHAR(64) NULL AFTER daily_date;

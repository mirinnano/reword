-- Learning content (phase 2). Rows are written only by `content import`,
-- which syncs them from the YAML files under content/.

-- Kinds of question (大問). The registry in internal/content is the source
-- of truth; import upserts these rows.
CREATE TABLE question_types (
  id         SMALLINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  code       VARCHAR(32)  NOT NULL,
  grade_id   TINYINT UNSIGNED NOT NULL,
  name_ja    VARCHAR(64)  NOT NULL,
  format     ENUM('mcq') NOT NULL,
  sort_order SMALLINT UNSIGNED NOT NULL,
  UNIQUE KEY uq_question_types_code (code),
  CONSTRAINT fk_question_types_grade FOREIGN KEY (grade_id) REFERENCES grades (id)
) ENGINE=InnoDB;

-- Questions are versioned and never edited after insert: changing the YAML
-- adds version N+1 and retires N, so recorded answers keep pointing at the
-- exact text and answer the learner saw.
CREATE TABLE questions (
  id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  stable_key       VARCHAR(64)  NOT NULL,
  version          INT UNSIGNED NOT NULL,
  question_type_id SMALLINT UNSIGNED NOT NULL,
  -- mcq: {"stem": "...", "choices": ["..", "..", "..", ".."]}
  payload          JSON         NOT NULL CHECK (JSON_VALID(payload)),
  -- mcq: {"index": 0}
  answer           JSON         NOT NULL CHECK (JSON_VALID(answer)),
  explanation_ja   TEXT         NOT NULL,
  difficulty       TINYINT UNSIGNED NOT NULL,
  -- Related vocab keys, as a JSON array of strings.
  vocab_keys       JSON         NOT NULL CHECK (JSON_VALID(vocab_keys)),
  content_hash     BINARY(32)   NOT NULL,
  status           ENUM('published','retired') NOT NULL,
  created_at       DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY uq_questions_key_version (stable_key, version),
  KEY idx_questions_type_status (question_type_id, status),
  CONSTRAINT fk_questions_type FOREIGN KEY (question_type_id) REFERENCES question_types (id)
) ENGINE=InnoDB;

-- Vocabulary is updated in place: fixing a meaning should not reset anyone's
-- review history, which will hang off stable_key.
CREATE TABLE vocab_items (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  stable_key   VARCHAR(64)  NOT NULL,
  grade_id     TINYINT UNSIGNED NOT NULL,
  headword     VARCHAR(100) NOT NULL,
  pos          ENUM('noun','verb','adjective','adverb','phrase','other') NOT NULL,
  -- {"meanings_ja": [...], "example_en": "...", "example_ja": "..."}
  payload      JSON         NOT NULL CHECK (JSON_VALID(payload)),
  content_hash BINARY(32)   NOT NULL,
  status       ENUM('published','retired') NOT NULL,
  created_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  UNIQUE KEY uq_vocab_key (stable_key),
  KEY idx_vocab_grade_status (grade_id, status),
  CONSTRAINT fk_vocab_grade FOREIGN KEY (grade_id) REFERENCES grades (id)
) ENGINE=InnoDB;

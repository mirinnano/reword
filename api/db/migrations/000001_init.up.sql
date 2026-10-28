-- Exam structure: 英検 grades. Launch scope is pre2..g1; lower grades exist
-- in the enum so adding them later is a data change, not a schema change.
CREATE TABLE grades (
  id          TINYINT UNSIGNED PRIMARY KEY,
  code        VARCHAR(16)  NOT NULL UNIQUE,
  name_ja     VARCHAR(32)  NOT NULL,
  sort_order  TINYINT UNSIGNED NOT NULL,
  is_active   BOOLEAN NOT NULL DEFAULT FALSE
) ENGINE=InnoDB;

INSERT INTO grades (id, code, name_ja, sort_order, is_active) VALUES
  (1, 'g5',       '5級',       1, FALSE),
  (2, 'g4',       '4級',       2, FALSE),
  (3, 'g3',       '3級',       3, FALSE),
  (4, 'pre2',     '準2級',     4, TRUE),
  (5, 'pre2plus', '準2級プラス', 5, TRUE),
  (6, 'g2',       '2級',       6, TRUE),
  (7, 'pre1',     '準1級',     7, TRUE),
  (8, 'g1',       '1級',       8, TRUE);

-- DB-backed job queue (TTS generation, mail, rollups). Workers claim rows
-- with SELECT ... FOR UPDATE SKIP LOCKED, so no external broker is needed.
CREATE TABLE jobs (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  kind         VARCHAR(64)  NOT NULL,
  payload      JSON         NOT NULL CHECK (JSON_VALID(payload)),
  status       ENUM('queued','running','done','failed') NOT NULL DEFAULT 'queued',
  attempts     SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  max_attempts SMALLINT UNSIGNED NOT NULL DEFAULT 5,
  run_at       DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  locked_until DATETIME(6)  NULL,
  last_error   TEXT         NULL,
  created_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  KEY idx_jobs_claim (status, run_at)
) ENGINE=InnoDB;

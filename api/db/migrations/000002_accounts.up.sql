-- Accounts (phase 1). One login (users) owns up to 5 learner profiles that
-- the signed-in session switches between. Learners never log in separately.

CREATE TABLE users (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  -- Stored lowercased and trimmed by the application.
  email             VARCHAR(254) NOT NULL,
  password_hash     VARCHAR(255) NOT NULL,
  role              ENUM('member','editor','reviewer','admin') NOT NULL DEFAULT 'member',
  email_verified_at DATETIME(6)  NULL,
  created_at        DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at        DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  UNIQUE KEY uq_users_email (email)
) ENGINE=InnoDB;

CREATE TABLE learner_profiles (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id         BIGINT UNSIGNED NOT NULL,
  display_name    VARCHAR(40)  NOT NULL,
  target_grade_id TINYINT UNSIGNED NULL,
  created_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  KEY idx_profiles_user (user_id),
  CONSTRAINT fk_profiles_user  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_profiles_grade FOREIGN KEY (target_grade_id) REFERENCES grades (id)
) ENGINE=InnoDB;

-- Only SHA-256 of the session token is stored (ADR 0001).
CREATE TABLE sessions (
  token_hash        BINARY(32)   NOT NULL PRIMARY KEY,
  user_id           BIGINT UNSIGNED NOT NULL,
  active_profile_id BIGINT UNSIGNED NULL,
  ip                VARCHAR(45)  NOT NULL DEFAULT '',
  user_agent        VARCHAR(255) NOT NULL DEFAULT '',
  created_at        DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  last_seen_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  expires_at        DATETIME(6)  NOT NULL,
  KEY idx_sessions_user (user_id),
  KEY idx_sessions_expires (expires_at),
  CONSTRAINT fk_sessions_user    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_sessions_profile FOREIGN KEY (active_profile_id) REFERENCES learner_profiles (id) ON DELETE SET NULL
) ENGINE=InnoDB;

-- A signup creates no user. It creates a pending signup whose token is
-- emailed; the user is created only when that link is opened, with the
-- password from that same signup. This blocks account pre-hijacking (an
-- attacker registering someone else's address first with their own password).
CREATE TABLE pending_signups (
  token_hash      BINARY(32)   NOT NULL PRIMARY KEY,
  email           VARCHAR(254) NOT NULL,
  password_hash   VARCHAR(255) NOT NULL,
  display_name    VARCHAR(40)  NOT NULL,
  target_grade_id TINYINT UNSIGNED NULL,
  terms_version   VARCHAR(32)  NOT NULL,
  privacy_version VARCHAR(32)  NOT NULL,
  ip              VARCHAR(45)  NOT NULL DEFAULT '',
  expires_at      DATETIME(6)  NOT NULL,
  created_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  KEY idx_pending_signups_email (email),
  KEY idx_pending_signups_expires (expires_at)
) ENGINE=InnoDB;

-- Single-use tokens sent by email to existing users. Only the SHA-256 is stored.
CREATE TABLE email_tokens (
  token_hash BINARY(32)   NOT NULL PRIMARY KEY,
  user_id    BIGINT UNSIGNED NOT NULL,
  purpose    ENUM('reset_password') NOT NULL,
  expires_at DATETIME(6)  NOT NULL,
  used_at    DATETIME(6)  NULL,
  created_at DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  KEY idx_email_tokens_user (user_id, purpose),
  CONSTRAINT fk_email_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Which version of each legal document the account accepted, and when.
CREATE TABLE consents (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_id     BIGINT UNSIGNED NOT NULL,
  document    ENUM('terms','privacy') NOT NULL,
  version     VARCHAR(32)  NOT NULL,
  ip          VARCHAR(45)  NOT NULL DEFAULT '',
  accepted_at DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  KEY idx_consents_user (user_id, document),
  CONSTRAINT fk_consents_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Fixed-window counters for rate limiting (login, signup, password reset).
CREATE TABLE rate_limits (
  bucket       VARCHAR(191) NOT NULL,
  window_start DATETIME     NOT NULL,
  hits         INT UNSIGNED NOT NULL,
  PRIMARY KEY (bucket, window_start),
  KEY idx_rate_limits_window (window_start)
) ENGINE=InnoDB;

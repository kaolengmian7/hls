-- server/db/schema.sql
CREATE TABLE IF NOT EXISTS users (
  id           BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  mp_openid    VARCHAR(64)  NULL UNIQUE,
  open_openid  VARCHAR(64)  NULL UNIQUE,
  union_id     VARCHAR(64)  NULL UNIQUE,
  phone        VARCHAR(20)  NULL UNIQUE,
  nickname     VARCHAR(64)  NOT NULL DEFAULT '',
  avatar_url   VARCHAR(512) NOT NULL DEFAULT '',
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Run this migration on existing databases:
-- ALTER TABLE users ADD COLUMN phone VARCHAR(20) NULL UNIQUE AFTER union_id;

CREATE TABLE IF NOT EXISTS auth_states (
  state      VARCHAR(64) NOT NULL PRIMARY KEY,
  jwt        TEXT        NULL,    -- NULL until WeChat callback populates it (two-phase write)
  expires_at DATETIME    NOT NULL,
  INDEX idx_expires_at (expires_at)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS shop_rooms (
  code               VARCHAR(6)   NOT NULL PRIMARY KEY,
  status             ENUM('waiting','playing','closed') NOT NULL DEFAULT 'waiting',
  host_player_id     VARCHAR(20)  NOT NULL,
  guaranteed_amount  INT          NOT NULL DEFAULT 0,
  created_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_status (status)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS shop_room_players (
  id          BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  room_code   VARCHAR(6)   NOT NULL,
  user_id     BIGINT       NOT NULL,
  player_id   VARCHAR(20)  NOT NULL,
  role        VARCHAR(20)  NOT NULL DEFAULT '',
  coin        INT          NOT NULL DEFAULT 0,
  is_host     TINYINT(1)   NOT NULL DEFAULT 0,
  ready       TINYINT(1)   NOT NULL DEFAULT 0,
  left_at     DATETIME     NULL,
  joined_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_room_user (room_code, user_id),
  INDEX idx_room_code (room_code),
  INDEX idx_user_active (user_id, left_at)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS shop_lottery_records (
  id               BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  room_code        VARCHAR(6)   NOT NULL,
  player_id        VARCHAR(20)  NOT NULL,
  item_id          VARCHAR(64)  NOT NULL,
  item_name        VARCHAR(128) NOT NULL DEFAULT '',
  rarity           VARCHAR(32)  NOT NULL DEFAULT '',
  item_value       INT          NOT NULL DEFAULT 0,
  guarantee_before INT          NOT NULL DEFAULT 0,
  guarantee_after  INT          NOT NULL DEFAULT 0,
  created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_room (room_code)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS shop_settlement_records (
  id               VARCHAR(20)  NOT NULL PRIMARY KEY,
  room_code        VARCHAR(6)   NOT NULL,
  outcome          VARCHAR(20)  NOT NULL,
  guarantee_delta  INT          NOT NULL DEFAULT 0,
  guarantee_before INT          NOT NULL DEFAULT 0,
  guarantee_after  INT          NOT NULL DEFAULT 0,
  per_player       JSON         NOT NULL,
  created_at       DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_room (room_code)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS enforcer_accounts (
  user_id          BIGINT        PRIMARY KEY,
  deposit_required DECIMAL(10,2) NULL,
  deposit_paid     DECIMAL(10,2) NOT NULL DEFAULT 0,
  pending_fine     DECIMAL(10,2) NOT NULL DEFAULT 0,
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS payroll_items (
  id               BIGINT        AUTO_INCREMENT PRIMARY KEY,
  order_code       VARCHAR(20)   NOT NULL,
  enforcer_id      BIGINT        NOT NULL,
  enforcer_name    VARCHAR(50)   NOT NULL,
  status           ENUM('pending','paid') NOT NULL DEFAULT 'pending',
  gross            DECIMAL(10,2) NOT NULL DEFAULT 0,
  deposit_deducted DECIMAL(10,2) NOT NULL DEFAULT 0,
  fine_deducted    DECIMAL(10,2) NOT NULL DEFAULT 0,
  net_paid         DECIMAL(10,2) NOT NULL DEFAULT 0,
  source           ENUM('order','teaching') NOT NULL DEFAULT 'order',
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  paid_at          TIMESTAMP     NULL,
  UNIQUE KEY uq_order_enforcer (order_code, enforcer_id),
  INDEX idx_status (status)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS teaching_orders (
  id                BIGINT AUTO_INCREMENT PRIMARY KEY,
  code              VARCHAR(20) NOT NULL,
  title             VARCHAR(50) NOT NULL,
  student_id        BIGINT NOT NULL,
  student_name      VARCHAR(50) NOT NULL,
  dispatcher_id     BIGINT NOT NULL,
  dispatcher_name   VARCHAR(50) NOT NULL,
  teacher_id        BIGINT NULL,
  teacher_name      VARCHAR(50) NULL,
  price             DECIMAL(10,2) NOT NULL DEFAULT 0,
  commission_rate   DECIMAL(5,2)  NOT NULL DEFAULT 0,
  bonus_rate        DECIMAL(5,2)  NOT NULL DEFAULT 0,
  bonus_qualified   TINYINT(1)    NOT NULL DEFAULT 0,
  total_minutes     INT NOT NULL DEFAULT 0,
  remaining_minutes INT NOT NULL DEFAULT 0,
  status            ENUM('dispatching','ongoing','settled') NOT NULL DEFAULT 'dispatching',
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_code (code),
  INDEX idx_status (status),
  INDEX idx_teacher (teacher_id),
  INDEX idx_student (student_id)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS teaching_scorecards (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  duration_minutes    INT NOT NULL,
  scores              JSON NULL,
  comment             TEXT NULL,
  teacher_id          BIGINT NOT NULL,
  teacher_name        VARCHAR(50) NOT NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS teaching_time_logs (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  delta_minutes       INT NOT NULL,
  reason              ENUM('initial','topup','lesson') NOT NULL,
  ref_scorecard_id    BIGINT NULL,
  price_delta         DECIMAL(10,2) NOT NULL DEFAULT 0,
  operator_id         BIGINT NOT NULL,
  operator_name       VARCHAR(50) NOT NULL,
  operator_role       VARCHAR(20) NOT NULL,
  note                VARCHAR(255) NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS teaching_reviews (
  id                  BIGINT AUTO_INCREMENT PRIMARY KEY,
  teaching_order_code VARCHAR(20) NOT NULL,
  attitude_score      INT NOT NULL,
  technical_score     INT NOT NULL,
  comment             TEXT NULL,
  reviewer_id         BIGINT NOT NULL,
  reviewer_name       VARCHAR(50) NOT NULL,
  created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_order (teaching_order_code)
) DEFAULT CHARSET=utf8mb4;

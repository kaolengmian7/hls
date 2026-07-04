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

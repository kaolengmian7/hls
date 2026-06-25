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

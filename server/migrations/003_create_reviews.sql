-- server/migrations/003_create_reviews.sql
CREATE TABLE reviews (
  id                 BIGINT       AUTO_INCREMENT PRIMARY KEY,
  order_code         VARCHAR(20)  NOT NULL UNIQUE,
  attitude_score     TINYINT      NOT NULL,
  technical_score    TINYINT      NOT NULL,
  blame_boss         TINYINT(1)   NOT NULL DEFAULT 0,
  late               TINYINT(1)   NOT NULL DEFAULT 0,
  temp_unavailable   TINYINT(1)   NOT NULL DEFAULT 0,
  slow_reply         TINYINT(1)   NOT NULL DEFAULT 0,
  no_backpack        TINYINT(1)   NOT NULL DEFAULT 0,
  no_channel_card    TINYINT(1)   NOT NULL DEFAULT 0,
  no_supply_card     TINYINT(1)   NOT NULL DEFAULT 0,
  no_emotional_value TINYINT(1)   NOT NULL DEFAULT 0,
  grab_container     TINYINT(1)   NOT NULL DEFAULT 0,
  comment            TEXT         NULL,
  reviewer_id        BIGINT       NOT NULL,
  reviewer_name      VARCHAR(64)  NOT NULL,
  created_at         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

ALTER TABLE orders ADD COLUMN bonus_qualified TINYINT(1) NOT NULL DEFAULT 0 AFTER complaint_status;

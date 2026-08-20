-- 005_create_teaching.sql — teaching subsystem tables
CREATE TABLE teaching_orders (
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

CREATE TABLE teaching_scorecards (
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

CREATE TABLE teaching_time_logs (
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

CREATE TABLE teaching_reviews (
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

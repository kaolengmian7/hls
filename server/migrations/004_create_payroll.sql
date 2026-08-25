-- server/migrations/004_create_payroll.sql
CREATE TABLE enforcer_accounts (
  user_id          BIGINT        PRIMARY KEY,
  deposit_required DECIMAL(10,2) NULL,
  deposit_paid     DECIMAL(10,2) NOT NULL DEFAULT 0,
  pending_fine     DECIMAL(10,2) NOT NULL DEFAULT 0,
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) DEFAULT CHARSET=utf8mb4;

CREATE TABLE payroll_items (
  id               BIGINT        AUTO_INCREMENT PRIMARY KEY,
  order_code       VARCHAR(20)   NOT NULL,
  enforcer_id      BIGINT        NOT NULL,
  enforcer_name    VARCHAR(50)   NOT NULL,
  status           ENUM('pending','paid') NOT NULL DEFAULT 'pending',
  gross            DECIMAL(10,2) NOT NULL DEFAULT 0,
  deposit_deducted DECIMAL(10,2) NOT NULL DEFAULT 0,
  fine_deducted    DECIMAL(10,2) NOT NULL DEFAULT 0,
  net_paid         DECIMAL(10,2) NOT NULL DEFAULT 0,
  created_at       TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  paid_at          TIMESTAMP     NULL,
  UNIQUE KEY uq_order_enforcer (order_code, enforcer_id),
  INDEX idx_status (status)
) DEFAULT CHARSET=utf8mb4;

-- server/migrations/001_add_username_password_role.sql
ALTER TABLE users
  ADD COLUMN username      VARCHAR(7)   UNIQUE NULL,
  ADD COLUMN password_hash VARCHAR(255) NULL,
  ADD COLUMN role          ENUM('owner','service','enforcer','customer')
                           NOT NULL DEFAULT 'customer';

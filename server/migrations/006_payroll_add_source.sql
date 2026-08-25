-- 006_payroll_add_source.sql — distinguish order vs teaching payroll
ALTER TABLE payroll_items
  ADD COLUMN source ENUM('order','teaching') NOT NULL DEFAULT 'order';

-- Rollback 000103

ALTER TABLE shifts DROP COLUMN IF EXISTS capacity;

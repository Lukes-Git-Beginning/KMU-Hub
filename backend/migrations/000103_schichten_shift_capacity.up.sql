-- Migration 000103: Add capacity column to shifts table (Bug #8)
-- NULL means unlimited; positive integer caps the number of assignees.

ALTER TABLE shifts ADD COLUMN IF NOT EXISTS capacity INT NULL CHECK (capacity IS NULL OR capacity > 0);

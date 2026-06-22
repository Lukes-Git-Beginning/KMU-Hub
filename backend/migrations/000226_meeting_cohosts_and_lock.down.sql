-- Rollback Migration 000226

DROP TABLE IF EXISTS meeting_cohosts;

ALTER TABLE meetings DROP COLUMN IF EXISTS locked;

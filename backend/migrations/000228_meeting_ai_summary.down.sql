-- Rollback Migration 000228

ALTER TABLE meetings DROP COLUMN IF EXISTS ai_summary_at;
ALTER TABLE meetings DROP COLUMN IF EXISTS ai_summary;

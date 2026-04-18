-- Rollback Sprint 1 S1.R2.3
DROP INDEX IF EXISTS idx_recordings_started_by;

ALTER TABLE recordings
    DROP COLUMN IF EXISTS consent_snapshot,
    DROP COLUMN IF EXISTS started_by;

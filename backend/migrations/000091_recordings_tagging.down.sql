-- Rollback 000091: revert consent_snapshot NOT NULL constraint and comment.
-- The columns themselves (started_by, consent_snapshot) remain — they were
-- created in 000078 and are dropped only by 000078's down migration.

-- Remove the NOT NULL constraint (allow NULL again so 000078 down can drop cleanly)
ALTER TABLE recordings
    ALTER COLUMN consent_snapshot DROP NOT NULL,
    ALTER COLUMN consent_snapshot DROP DEFAULT;

-- Remove column comment
COMMENT ON COLUMN recordings.consent_snapshot IS NULL;

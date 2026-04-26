-- Sprint 2 Wave 1.B - S1.7 hardening of 000078:
-- Adds the NOT NULL guarantee to recordings.consent_snapshot via a
-- batched backfill + NOT VALID + VALIDATE pattern, so the migration
-- does not hold a long-running ACCESS EXCLUSIVE lock on the table.
--
-- Migration 000078 added started_by + consent_snapshot as nullable columns.
-- This migration sets the NOT NULL guarantee on consent_snapshot so every new
-- recording always carries an explicit JSON array (even if empty), and attaches
-- the forensic COMMENT required by R2-P0.3.
--
-- started_by already has an index from 000078 (idx_recordings_started_by).
-- We re-apply as IF NOT EXISTS to be idempotent.

-- 1. Add a default for any future inserts so the application code can rely on
--    consent_snapshot being non-null going forward. Default '[]'::jsonb is the
--    safe empty-array marker matching the Go marshalConsentSnapshot helper.
ALTER TABLE recordings ALTER COLUMN consent_snapshot SET DEFAULT '[]'::jsonb;

-- 2. Backfill existing rows in batches to avoid a long-running UPDATE that would
--    block all reads/writes against the recordings table for the duration of the
--    statement. 1000 rows per batch is safe even for tables with 1M+ rows.
DO $$
DECLARE
    updated_count INTEGER;
    total_updated INTEGER := 0;
BEGIN
    LOOP
        UPDATE recordings
        SET consent_snapshot = '[]'::jsonb
        WHERE id IN (
            SELECT id FROM recordings
            WHERE consent_snapshot IS NULL
            LIMIT 1000
            FOR UPDATE SKIP LOCKED
        );
        GET DIAGNOSTICS updated_count = ROW_COUNT;
        EXIT WHEN updated_count = 0;
        total_updated := total_updated + updated_count;
    END LOOP;
    RAISE NOTICE 'recordings consent_snapshot backfill: % rows', total_updated;
END$$;

-- 3. Add a CHECK constraint that asserts non-null but is NOT VALIDATED on existing
--    rows. Future inserts/updates that try to set NULL will be rejected immediately,
--    while existing rows are checked only when VALIDATE CONSTRAINT is run below.
--    Adding a NOT VALID constraint takes only a brief table-level lock — no full scan.
--    Guard against re-runs (idempotency) via pg_constraint lookup.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'recordings_consent_snapshot_not_null'
          AND conrelid = 'recordings'::regclass
    ) THEN
        ALTER TABLE recordings
            ADD CONSTRAINT recordings_consent_snapshot_not_null
            CHECK (consent_snapshot IS NOT NULL) NOT VALID;
    END IF;
END$$;

-- 4. Validate the constraint. After the batched backfill above, every row should
--    pass — but VALIDATE CONSTRAINT only acquires a SHARE UPDATE EXCLUSIVE lock
--    (concurrent reads/writes still allowed). Cheaper than ALTER TABLE ... SET NOT NULL.
ALTER TABLE recordings VALIDATE CONSTRAINT recordings_consent_snapshot_not_null;

-- Forensic comment (explains what lives in the JSONB for auditors / DSGVO)
COMMENT ON COLUMN recordings.consent_snapshot IS
    'Frozen consent record at recording start: [{"user_id":"...","display_name":"...","joined_at":"..."}]. '
    'Provides an immutable forensic trail per R2-P0.3 (UWG §7 / DSGVO Art. 6). '
    'Written by the recording service at CreateRecording time and never mutated.';

-- Ensure index exists (idempotent guard in case 000078 was partially applied)
CREATE INDEX IF NOT EXISTS idx_recordings_started_by ON recordings(started_by) WHERE started_by IS NOT NULL;

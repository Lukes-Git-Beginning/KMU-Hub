-- Migration 000091: Recording tagging hardening.
--
-- Migration 000078 added started_by + consent_snapshot as nullable columns.
-- This migration sets the NOT NULL default on consent_snapshot so every new
-- recording always carries an explicit JSON array (even if empty), and attaches
-- the forensic COMMENT required by R2-P0.3.
--
-- started_by already has an index from 000078 (idx_recordings_started_by).
-- We re-apply as IF NOT EXISTS to be idempotent.

-- Backfill rows inserted before this migration (consent_snapshot still NULL)
UPDATE recordings SET consent_snapshot = '[]' WHERE consent_snapshot IS NULL;

-- Tighten the constraint: disallow future NULLs
ALTER TABLE recordings
    ALTER COLUMN consent_snapshot SET NOT NULL,
    ALTER COLUMN consent_snapshot SET DEFAULT '[]';

-- Forensic comment (explains what lives in the JSONB for auditors / DSGVO)
COMMENT ON COLUMN recordings.consent_snapshot IS
    'Frozen consent record at recording start: [{"user_id":"...","display_name":"...","joined_at":"..."}]. '
    'Provides an immutable forensic trail per R2-P0.3 (UWG §7 / DSGVO Art. 6). '
    'Written by the recording service at CreateRecording time and never mutated.';

-- Ensure index exists (idempotent guard in case 000078 was partially applied)
CREATE INDEX IF NOT EXISTS idx_recordings_started_by ON recordings(started_by) WHERE started_by IS NOT NULL;

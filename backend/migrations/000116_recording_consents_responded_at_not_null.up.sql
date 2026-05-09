-- Migration 000107 added recording_consents.responded_at as NULLABLE, but
-- Service.SetConsent (backend/internal/work/recording/service.go:229) has always
-- written time.Now() unconditionally. The NULLABLE column hid this contract
-- from Postgres and from any future direct INSERTs.
--
-- Idempotent backfill: any row predating 000107 (or written by a hypothetical
-- direct-SQL bypass) gets the current timestamp. On Welle-3-or-newer data this
-- is a no-op.
UPDATE recording_consents
SET responded_at = NOW()
WHERE responded_at IS NULL;

ALTER TABLE recording_consents
    ALTER COLUMN responded_at SET NOT NULL;

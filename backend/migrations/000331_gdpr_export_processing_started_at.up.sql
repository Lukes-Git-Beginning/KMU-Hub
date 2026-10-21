-- Lease timestamp for the export sweeper: set when generation starts, kept on a
-- failed run so the retry waits one lease from the last attempt.
ALTER TABLE gdpr_export_requests ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ;

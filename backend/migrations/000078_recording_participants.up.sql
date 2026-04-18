-- Sprint 1 S1.R2.3: Add started_by and consent_snapshot to recordings.
-- started_by: who triggered the recording (audit trail, DSGVO)
-- consent_snapshot: JSON array of participants at recording start (immutable DSGVO record)
ALTER TABLE recordings
    ADD COLUMN started_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN consent_snapshot JSONB NULL;

CREATE INDEX idx_recordings_started_by ON recordings(started_by) WHERE started_by IS NOT NULL;

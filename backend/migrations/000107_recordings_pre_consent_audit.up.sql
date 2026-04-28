ALTER TABLE recordings
    ADD COLUMN IF NOT EXISTS pre_recording_consent_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS initiator_consent_id UUID NULL;

ALTER TABLE recording_consents
    ADD COLUMN IF NOT EXISTS responded_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_recordings_pre_consent
    ON recordings(initiator_consent_id) WHERE initiator_consent_id IS NOT NULL;

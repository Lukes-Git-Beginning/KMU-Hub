DROP INDEX IF EXISTS idx_recordings_pre_consent;

ALTER TABLE recording_consents
    DROP COLUMN IF EXISTS responded_at;

ALTER TABLE recordings
    DROP COLUMN IF EXISTS initiator_consent_id,
    DROP COLUMN IF EXISTS pre_recording_consent_at;

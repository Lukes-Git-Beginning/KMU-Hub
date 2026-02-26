-- Revert guest chat support

ALTER TABLE messages DROP CONSTRAINT IF EXISTS chk_message_sender;
DROP INDEX IF EXISTS idx_messages_guest_session;
ALTER TABLE messages DROP COLUMN IF EXISTS guest_session_id;
ALTER TABLE messages ALTER COLUMN created_by SET NOT NULL;

ALTER TABLE channels DROP COLUMN IF EXISTS is_guest_enabled;

DROP TABLE IF EXISTS guest_channel_config;
DROP TABLE IF EXISTS guest_sessions;

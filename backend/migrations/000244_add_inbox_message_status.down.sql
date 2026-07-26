BEGIN;

DROP INDEX IF EXISTS idx_inbox_messages_user_status;
ALTER TABLE inbox_messages DROP COLUMN IF EXISTS status;

COMMIT;

DROP INDEX IF EXISTS idx_tickets_source_message;

ALTER TABLE tickets
    DROP COLUMN IF EXISTS source_message_id,
    DROP COLUMN IF EXISTS source_channel;

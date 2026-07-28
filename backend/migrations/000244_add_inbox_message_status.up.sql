-- Adds the conversation-level status (open/pending/resolved/closed) that the
-- FE previously overlaid client-side in stores/inboxStatus.ts (backend-gaps.md
-- line 454: "kein Feld in InboxMessage/proto").

BEGIN;

ALTER TABLE inbox_messages
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'pending', 'resolved', 'closed'));

-- Fast lookup: user's inbox filtered by status (mirrors idx_inbox_messages_user_channel).
CREATE INDEX idx_inbox_messages_user_status ON inbox_messages(user_id, status, received_at DESC)
    WHERE is_archived = false;

COMMIT;

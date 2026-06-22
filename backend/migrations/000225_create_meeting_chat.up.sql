-- Migration 000225: Create meeting_chat table for persisted in-call chat messages.
-- Chat messages are append-only (no UPDATE), tenant-scoped via RLS.

CREATE TABLE meeting_chat (
    id          UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    meeting_id  UUID        NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    sender_id   UUID        NOT NULL,
    sender_name TEXT        NOT NULL DEFAULT '',
    message     TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT meeting_chat_pkey PRIMARY KEY (id)
);

CREATE INDEX meeting_chat_meeting_created_idx ON meeting_chat (meeting_id, created_at);

CALL enable_tenant_rls('meeting_chat');

-- Migration 000237: inbox_thread_messages
-- Persists the conversation thread per inbox message: the materialized inbound
-- original, outbound replies, and internal notes. Reply now writes an outbound row
-- here so the thread is durable instead of FE-synthetic.

BEGIN;

CREATE TABLE inbox_thread_messages (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message_id  UUID        NOT NULL REFERENCES inbox_messages(id) ON DELETE CASCADE,
    -- NULL author_id = inbound/external message (author_name holds the sender).
    author_id   UUID,
    author_name TEXT,
    direction   TEXT        NOT NULL CHECK (direction IN ('inbound','outbound','internal')),
    body        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inbox_thread_message ON inbox_thread_messages (message_id, created_at);

-- RLS: standard tenant isolation (matches pattern in 000232).
ALTER TABLE inbox_thread_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE inbox_thread_messages FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON inbox_thread_messages
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;

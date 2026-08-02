-- Migration 000262: message_bookmarks -- per-user saved chat messages.
--
-- Mirrors the shape of message_reactions (migration 000038/000115/000122):
-- composite PK on (user_id, message_id) since a user can bookmark a given
-- message at most once, tenant_id NOT NULL from the start because this is a
-- new table, not a retrofit.

BEGIN;

CREATE TABLE message_bookmarks (
    tenant_id  UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    message_id UUID        NOT NULL REFERENCES messages (id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, message_id)
);

CREATE INDEX idx_message_bookmarks_message ON message_bookmarks (message_id);

CALL enable_tenant_rls('message_bookmarks');

COMMIT;

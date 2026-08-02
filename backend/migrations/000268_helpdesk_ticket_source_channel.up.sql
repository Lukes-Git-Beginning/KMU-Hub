-- ============================================================================
-- Multi-channel ticket provenance (backend-gaps.md §helpdesk "source_channel
-- ins Ticket-Modell + Inbox->Ticket-Adapter"). A ticket created from an inbox
-- message carries source_channel (mirrors inbox_messages.channel) and a
-- reference to the originating message -- not a second copy of it. Both are
-- nullable: a ticket created directly (not via the adapter) has neither.
-- ============================================================================

ALTER TABLE tickets
    ADD COLUMN source_channel VARCHAR(20) NULL
        CHECK (source_channel IN ('email', 'chat', 'notification')),
    ADD COLUMN source_message_id UUID NULL REFERENCES inbox_messages(id) ON DELETE SET NULL;

-- Idempotency backstop: a given inbox message maps to at most one ticket,
-- enforced at the DB level in case two concurrent conversions race past the
-- application-level pre-check.
CREATE UNIQUE INDEX idx_tickets_source_message ON tickets (source_message_id)
    WHERE source_message_id IS NOT NULL;

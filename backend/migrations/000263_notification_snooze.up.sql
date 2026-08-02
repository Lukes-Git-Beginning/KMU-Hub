-- Migration 000263: Add snoozed_until to notifications
-- A snoozed notification is hidden from the list and unread count until the
-- given time passes (matches the inbox module's snooze semantics).

ALTER TABLE notifications
    ADD COLUMN snoozed_until TIMESTAMPTZ;

-- Partial index for fast lookup of currently snoozed notifications per user/tenant
CREATE INDEX idx_notifications_snoozed
    ON notifications (tenant_id, user_id)
    WHERE snoozed_until IS NOT NULL;

-- Migration 000229: Add is_pinned, is_dismissed, actor_name to notifications
-- These fields enable the frontend to pin/dismiss notifications and display actor names.

ALTER TABLE notifications
    ADD COLUMN is_pinned    BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN is_dismissed BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN actor_name   TEXT;

-- Partial index for fast lookup of pinned notifications per user/tenant
CREATE INDEX idx_notifications_pinned
    ON notifications (tenant_id, user_id)
    WHERE is_pinned = true;

-- Partial index for filtered list (exclude dismissed)
CREATE INDEX idx_notifications_not_dismissed
    ON notifications (tenant_id, user_id, created_at DESC)
    WHERE is_dismissed = false;

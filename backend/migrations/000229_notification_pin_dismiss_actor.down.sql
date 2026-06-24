-- Rollback Migration 000229

DROP INDEX IF EXISTS idx_notifications_not_dismissed;
DROP INDEX IF EXISTS idx_notifications_pinned;

ALTER TABLE notifications
    DROP COLUMN IF EXISTS actor_name,
    DROP COLUMN IF EXISTS is_dismissed,
    DROP COLUMN IF EXISTS is_pinned;

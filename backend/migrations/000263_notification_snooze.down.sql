-- Rollback Migration 000263

DROP INDEX IF EXISTS idx_notifications_snoozed;

ALTER TABLE notifications
    DROP COLUMN IF EXISTS snoozed_until;

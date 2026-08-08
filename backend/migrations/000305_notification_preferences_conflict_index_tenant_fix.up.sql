-- Migration 000124 made tenant_id NOT NULL on notification_preferences but never
-- widened the event-type unique index to include it. postgres_repository.go's
-- UpsertPreference has been targeting `ON CONFLICT (tenant_id, user_id,
-- event_type_key)` since before that migration, which does not match this
-- 2-column partial index -- every upsert of an event-type-scoped preference
-- has therefore failed with "no unique or exclusion constraint matching the ON
-- CONFLICT specification" since 000124 shipped. Widening the index (superset of
-- the old one, so it cannot conflict with any existing row) fixes the write path.
DROP INDEX idx_notification_preferences_event_type;
CREATE UNIQUE INDEX idx_notification_preferences_event_type
    ON notification_preferences(tenant_id, user_id, event_type_key)
    WHERE event_type_key IS NOT NULL;

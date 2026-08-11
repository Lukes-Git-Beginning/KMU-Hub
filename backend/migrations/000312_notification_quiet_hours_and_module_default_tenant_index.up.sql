-- Migration 000312: Widen the two remaining notification unique indexes to carry tenant_id.
--
-- Migration 000124 made tenant_id NOT NULL on notification_quiet_hours and
-- notification_preferences but only migration 000305 ever widened one of the
-- unique indexes to match (idx_notification_preferences_event_type). Two write
-- paths were left broken by the same drift:
--
--   1. UpsertQuietHours targets `ON CONFLICT (tenant_id, user_id)`, but
--      notification_quiet_hours only carries the inline `UNIQUE(user_id)` from
--      migration 000022. Postgres validates the ON CONFLICT arbiter at PLAN
--      time, so EVERY call fails with SQLSTATE 42P10 -- quiet hours cannot be
--      configured at all.
--   2. Module defaults (event_type_key IS NULL) can only be arbitrated by
--      idx_notification_preferences_module_default, which is still scoped
--      (user_id, module_id). UpsertPreference can therefore not name it in an
--      ON CONFLICT clause that also scopes by tenant, and a second upsert of
--      the same module default fails with SQLSTATE 23505 instead of updating.
--
-- Both new indexes are supersets of the ones they replace, so they cannot
-- collide with any existing row.

ALTER TABLE notification_quiet_hours
    DROP CONSTRAINT notification_quiet_hours_user_id_key;

CREATE UNIQUE INDEX idx_notification_quiet_hours_user
    ON notification_quiet_hours(tenant_id, user_id);

DROP INDEX idx_notification_preferences_module_default;
CREATE UNIQUE INDEX idx_notification_preferences_module_default
    ON notification_preferences(tenant_id, user_id, module_id)
    WHERE event_type_key IS NULL AND module_id IS NOT NULL;

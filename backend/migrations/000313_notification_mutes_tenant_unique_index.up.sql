-- Migration 000313: Widen the notification_mutes unique constraint to carry tenant_id.
--
-- Same index drift as migrations 000305 and 000312, for the last remaining
-- table of the notification schema: notification_mutes still carries the
-- inline UNIQUE(user_id, module_id, resource_id) from migration 000022, while
-- migration 000110 added tenant_id and 000124 made it NOT NULL and enabled RLS.
--
-- The write path fails differently than the two fixed before it. CreateMute
-- names no ON CONFLICT arbiter, so there is no 42P10 planning error. Instead
-- Service.MuteResource checks IsResourceMuted first, which is tenant-scoped
-- and therefore cannot see another tenant's row, and then lets CreateMute run
-- straight into the tenant-less unique constraint with SQLSTATE 23505. The
-- caller gets a raw 500 instead of either a successful mute or
-- ErrMuteAlreadyExists.
--
-- The new index is a superset of the one it replaces and cannot collide with
-- any existing row.

ALTER TABLE notification_mutes
    DROP CONSTRAINT notification_mutes_user_id_module_id_resource_id_key;

CREATE UNIQUE INDEX idx_notification_mutes_resource
    ON notification_mutes(tenant_id, user_id, module_id, resource_id);

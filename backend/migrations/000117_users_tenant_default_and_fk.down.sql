-- Reverse 000117_users_tenant_default_and_fk.
-- The backfill is intentionally NOT reverted — restoring orphaned Zero-UUID
-- rows would re-introduce the original bug. Down only drops the FK so the
-- table reverts to its pre-FK state for emergency rollback scenarios.

BEGIN;

ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_tenant;

COMMIT;

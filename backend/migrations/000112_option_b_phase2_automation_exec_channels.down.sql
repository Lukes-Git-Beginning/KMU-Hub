-- Rollback: drop tenant_id columns added in 000112
-- Reversed order (last added first). DROP COLUMN removes associated indexes automatically.
-- NOTE: NOT NULL constraint is dropped implicitly with the column.

ALTER TABLE lexware_sync_configs DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE bexio_sync_configs DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE integration_configs DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE channel_memberships DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE automation_executions DROP COLUMN IF EXISTS tenant_id;

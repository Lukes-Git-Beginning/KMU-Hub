-- Option-B Phase 2: add tenant_id to automation_executions, channel_memberships,
-- and integration config tables.
-- Sentinel: 00000000-0000-0000-0000-000000000001 (existing single-tenant data)
-- All ALTERs use ADD COLUMN IF NOT EXISTS to be idempotent.
-- NOTE: automations and channels already have tenant_id from 000106.

-- automation_executions
ALTER TABLE automation_executions ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- JOIN-Backfill from parent automations table (000106 gave automations tenant_id)
UPDATE automation_executions ae
SET tenant_id = a.tenant_id
FROM automations a
WHERE ae.automation_id = a.id
  AND ae.tenant_id IS NULL;

-- NOT NULL is safe: backfill above guarantees every row with a valid automation_id is filled;
-- orphaned rows (ON DELETE CASCADE) cannot exist due to the FK constraint.
ALTER TABLE automation_executions ALTER COLUMN tenant_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_automation_executions_tenant_id ON automation_executions(tenant_id);

-- channel_memberships
ALTER TABLE channel_memberships ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- JOIN-Backfill from parent channels table (000106 gave channels tenant_id)
UPDATE channel_memberships cm
SET tenant_id = c.tenant_id
FROM channels c
WHERE cm.channel_id = c.id
  AND cm.tenant_id IS NULL;

-- NOT NULL is safe: FK ON DELETE CASCADE prevents orphaned rows; backfill covers all existing.
ALTER TABLE channel_memberships ALTER COLUMN tenant_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_channel_memberships_tenant_id ON channel_memberships(tenant_id);

-- integration_configs
-- No backfill source available (platform-scoped, not user-scoped); stays nullable until app sets it.
ALTER TABLE integration_configs ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_integration_configs_tenant_id ON integration_configs(tenant_id);

-- bexio_sync_configs
-- Linked to integration_configs via config_id; tenant derivable at app level.
ALTER TABLE bexio_sync_configs ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_bexio_sync_configs_tenant_id ON bexio_sync_configs(tenant_id);

-- lexware_sync_configs
-- Linked to integration_configs via config_id; tenant derivable at app level.
ALTER TABLE lexware_sync_configs ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_lexware_sync_configs_tenant_id ON lexware_sync_configs(tenant_id);

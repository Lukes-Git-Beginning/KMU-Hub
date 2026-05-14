-- Reverse 000125: deactivate RLS, restore UNIQUE(platform), drop tenant FKs,
-- relax tenant_id back to NULLABLE.

BEGIN;

DROP POLICY IF EXISTS tenant_isolation ON lexware_sync_configs;
ALTER TABLE lexware_sync_configs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE lexware_sync_configs DISABLE ROW LEVEL SECURITY;
ALTER TABLE lexware_sync_configs DROP CONSTRAINT IF EXISTS fk_lexware_sync_configs_tenant;
ALTER TABLE lexware_sync_configs ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON bexio_sync_configs;
ALTER TABLE bexio_sync_configs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE bexio_sync_configs DISABLE ROW LEVEL SECURITY;
ALTER TABLE bexio_sync_configs DROP CONSTRAINT IF EXISTS fk_bexio_sync_configs_tenant;
ALTER TABLE bexio_sync_configs ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON integration_configs;
ALTER TABLE integration_configs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE integration_configs DISABLE ROW LEVEL SECURITY;

ALTER TABLE integration_configs DROP CONSTRAINT IF EXISTS uq_integration_configs_platform_tenant;
ALTER TABLE integration_configs
    ADD CONSTRAINT integration_configs_platform_key UNIQUE (platform);

ALTER TABLE integration_configs DROP CONSTRAINT IF EXISTS fk_integration_configs_tenant;
ALTER TABLE integration_configs ALTER COLUMN tenant_id DROP NOT NULL;

COMMIT;

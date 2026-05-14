-- Sprint 4 Welle 4.1 — RLS Welle 4a: integration_configs constraint swap + RLS for the
-- 3 tables explicitly deferred from Welle 3 (see 000124 header comment).
-- 2026-05-14
--
-- The three target tables share one problem: the legacy single-tenant schema
-- enforces UNIQUE(platform) on integration_configs, which is incompatible with
-- per-tenant rows. RLS cannot be activated until that constraint is replaced
-- with UNIQUE(platform, tenant_id). The two child tables (bexio_sync_configs,
-- lexware_sync_configs) have a NULLABLE tenant_id column since mig 000112 and
-- can be backfilled via the integration_configs FK once the parent is settled.
--
-- Order matters:
--   1. integration_configs: backfill sentinel, SET NOT NULL, ADD FK, DROP old
--      UNIQUE(platform), ADD UNIQUE(platform, tenant_id)
--   2. bexio_sync_configs: backfill via config_id JOIN, SET NOT NULL, ADD FK
--   3. lexware_sync_configs: idem
--   4. CALL enable_tenant_rls() on all three
--
-- Sentinel: 00000000-0000-0000-0000-000000000001 (existing single-tenant data)

SET LOCAL row_security = off;

BEGIN;

-- ============================================================================
-- 1) integration_configs — backfill + constraint swap
-- ============================================================================

UPDATE integration_configs SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE integration_configs ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE integration_configs
    ADD CONSTRAINT fk_integration_configs_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;

ALTER TABLE integration_configs DROP CONSTRAINT IF EXISTS integration_configs_platform_key;
ALTER TABLE integration_configs
    ADD CONSTRAINT uq_integration_configs_platform_tenant
    UNIQUE (platform, tenant_id);

-- ============================================================================
-- 2) bexio_sync_configs — backfill via integration_configs.tenant_id
-- ============================================================================

UPDATE bexio_sync_configs bsc
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE bsc.config_id = ic.id
  AND bsc.tenant_id IS NULL;
-- orphaned rows (config deleted without cascade) get sentinel
UPDATE bexio_sync_configs SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE bexio_sync_configs ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bexio_sync_configs
    ADD CONSTRAINT fk_bexio_sync_configs_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;

-- ============================================================================
-- 3) lexware_sync_configs — backfill via integration_configs.tenant_id
-- ============================================================================

UPDATE lexware_sync_configs lsc
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE lsc.config_id = ic.id
  AND lsc.tenant_id IS NULL;
UPDATE lexware_sync_configs SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE lexware_sync_configs ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE lexware_sync_configs
    ADD CONSTRAINT fk_lexware_sync_configs_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;

-- ============================================================================
-- 4) Activate RLS on all three
-- ============================================================================

CALL enable_tenant_rls('integration_configs');
CALL enable_tenant_rls('bexio_sync_configs');
CALL enable_tenant_rls('lexware_sync_configs');

COMMIT;

DROP INDEX IF EXISTS idx_bexio_sync_log_config;
DROP TABLE IF EXISTS bexio_sync_log;
DROP TABLE IF EXISTS bexio_field_mappings;
DROP INDEX IF EXISTS idx_bexio_entity_mappings_bexio;
DROP INDEX IF EXISTS idx_bexio_entity_mappings_kmuhub;
DROP TABLE IF EXISTS bexio_entity_mappings;
DROP TABLE IF EXISTS bexio_sync_configs;

-- Restore original platform CHECK constraint (without 'bexio')
ALTER TABLE integration_configs DROP CONSTRAINT IF EXISTS integration_configs_platform_check;
ALTER TABLE integration_configs ADD CONSTRAINT integration_configs_platform_check
    CHECK (platform IN ('teams', 'slack'));

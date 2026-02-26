-- Drop DATEV upload tables
DROP TABLE IF EXISTS datev_upload_log;
DROP TABLE IF EXISTS datev_upload_configs;

-- Drop Lexware tables
DROP TABLE IF EXISTS lexware_webhook_subscriptions;
DROP TABLE IF EXISTS lexware_sync_log;
DROP TABLE IF EXISTS lexware_field_mappings;
DROP TABLE IF EXISTS lexware_entity_mappings;
DROP TABLE IF EXISTS lexware_sync_configs;

-- Revert integration_configs platform CHECK to exclude 'lexware' and 'datev_api'
ALTER TABLE integration_configs DROP CONSTRAINT IF EXISTS integration_configs_platform_check;
ALTER TABLE integration_configs ADD CONSTRAINT integration_configs_platform_check
    CHECK (platform IN ('teams', 'slack', 'bexio'));

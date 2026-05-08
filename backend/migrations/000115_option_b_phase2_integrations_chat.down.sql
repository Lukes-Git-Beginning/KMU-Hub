-- Reverse of 000115: drop tenant_id columns, constraints and indexes from
-- integration-mapping, CalDAV-sync, and chat/call tables.

-- ============================================================================
-- GROUP F: Chat / Presence / Call
-- ============================================================================
ALTER TABLE guest_channel_config   DROP CONSTRAINT IF EXISTS fk_guest_channel_config_tenant;
DROP INDEX IF EXISTS idx_guest_channel_config_tenant_id;
ALTER TABLE guest_channel_config   DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE guest_sessions         DROP CONSTRAINT IF EXISTS fk_guest_sessions_tenant;
DROP INDEX IF EXISTS idx_guest_sessions_tenant_id;
ALTER TABLE guest_sessions         DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE call_participants      DROP CONSTRAINT IF EXISTS fk_call_participants_tenant;
DROP INDEX IF EXISTS idx_call_participants_tenant_id;
ALTER TABLE call_participants      DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE call_sessions          DROP CONSTRAINT IF EXISTS fk_call_sessions_tenant;
DROP INDEX IF EXISTS idx_call_sessions_tenant_id;
ALTER TABLE call_sessions          DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE message_mentions       DROP CONSTRAINT IF EXISTS fk_message_mentions_tenant;
DROP INDEX IF EXISTS idx_message_mentions_tenant_id;
ALTER TABLE message_mentions       DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE message_reactions      DROP CONSTRAINT IF EXISTS fk_message_reactions_tenant;
DROP INDEX IF EXISTS idx_message_reactions_tenant_id;
ALTER TABLE message_reactions      DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE chat_files             DROP CONSTRAINT IF EXISTS fk_chat_files_tenant;
DROP INDEX IF EXISTS idx_chat_files_tenant_id;
ALTER TABLE chat_files             DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- GROUP E: Integration Mappings
-- ============================================================================
DROP INDEX IF EXISTS idx_integration_link_tokens_tenant_id;
ALTER TABLE integration_link_tokens        DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE integration_delivery_log       DROP CONSTRAINT IF EXISTS fk_integration_delivery_log_tenant;
DROP INDEX IF EXISTS idx_integration_delivery_log_tenant_id;
ALTER TABLE integration_delivery_log       DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE integration_account_links      DROP CONSTRAINT IF EXISTS fk_integration_account_links_tenant;
DROP INDEX IF EXISTS idx_integration_account_links_tenant_id;
ALTER TABLE integration_account_links      DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE integration_channel_mappings   DROP CONSTRAINT IF EXISTS fk_integration_channel_mappings_tenant;
DROP INDEX IF EXISTS idx_integration_channel_mappings_tenant_id;
ALTER TABLE integration_channel_mappings   DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE datev_upload_log               DROP CONSTRAINT IF EXISTS fk_datev_upload_log_tenant;
DROP INDEX IF EXISTS idx_datev_upload_log_tenant_id;
ALTER TABLE datev_upload_log               DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE datev_upload_configs           DROP CONSTRAINT IF EXISTS fk_datev_upload_configs_tenant;
DROP INDEX IF EXISTS idx_datev_upload_configs_tenant_id;
ALTER TABLE datev_upload_configs           DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE lexware_webhook_subscriptions  DROP CONSTRAINT IF EXISTS fk_lexware_webhook_subscriptions_tenant;
DROP INDEX IF EXISTS idx_lexware_webhook_subscriptions_tenant_id;
ALTER TABLE lexware_webhook_subscriptions  DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE lexware_sync_log               DROP CONSTRAINT IF EXISTS fk_lexware_sync_log_tenant;
DROP INDEX IF EXISTS idx_lexware_sync_log_tenant_id;
ALTER TABLE lexware_sync_log               DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE lexware_field_mappings         DROP CONSTRAINT IF EXISTS fk_lexware_field_mappings_tenant;
DROP INDEX IF EXISTS idx_lexware_field_mappings_tenant_id;
ALTER TABLE lexware_field_mappings         DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE lexware_entity_mappings        DROP CONSTRAINT IF EXISTS fk_lexware_entity_mappings_tenant;
DROP INDEX IF EXISTS idx_lexware_entity_mappings_tenant_id;
ALTER TABLE lexware_entity_mappings        DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE bexio_sync_log                 DROP CONSTRAINT IF EXISTS fk_bexio_sync_log_tenant;
DROP INDEX IF EXISTS idx_bexio_sync_log_tenant_id;
ALTER TABLE bexio_sync_log                 DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE bexio_field_mappings           DROP CONSTRAINT IF EXISTS fk_bexio_field_mappings_tenant;
DROP INDEX IF EXISTS idx_bexio_field_mappings_tenant_id;
ALTER TABLE bexio_field_mappings           DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE bexio_entity_mappings          DROP CONSTRAINT IF EXISTS fk_bexio_entity_mappings_tenant;
DROP INDEX IF EXISTS idx_bexio_entity_mappings_tenant_id;
ALTER TABLE bexio_entity_mappings          DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- GROUP D-REST: CalDAV Sync Layer
-- ============================================================================
ALTER TABLE caldav_change_log              DROP CONSTRAINT IF EXISTS fk_caldav_change_log_tenant;
DROP INDEX IF EXISTS idx_caldav_change_log_tenant_id;
ALTER TABLE caldav_change_log              DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE caldav_sync_versions           DROP CONSTRAINT IF EXISTS fk_caldav_sync_versions_tenant;
DROP INDEX IF EXISTS idx_caldav_sync_versions_tenant_id;
ALTER TABLE caldav_sync_versions           DROP COLUMN IF EXISTS tenant_id;

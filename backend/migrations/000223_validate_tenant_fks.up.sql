-- Migration 000223: Validate deferred tenant_id foreign keys.
--
-- Purpose: ~29 FKs on tenant_id → tenants(id) were created with NOT VALID in
-- migrations 000114, 000115, 000125, and 000126. NOT VALID skips the row scan
-- at constraint-creation time, meaning pre-existing rows are NOT guaranteed to
-- reference a valid tenants.id row. This migration validates all of them.
--
-- golang-migrate runs each migration atomically (implicit transaction). If any
-- VALIDATE CONSTRAINT fails (e.g. orphaned rows detected), the entire migration
-- is rolled back and the schema version stays at 222. Fix the orphans first,
-- then re-run migrate up.
--
-- CD-safety: VALIDATE CONSTRAINT acquires only a SHARE UPDATE EXCLUSIVE lock
-- (does NOT block reads or writes on the table). Safe to run in production
-- without a maintenance window under normal load.

-- ============================================================================
-- ORPHAN GUARD
-- Check all affected tables for rows whose tenant_id does not exist in tenants.
-- Accumulates all violations into a single error message so that one migration
-- run reveals ALL problematic tables at once.
-- ============================================================================
DO $$
DECLARE
    _tables  TEXT[] := ARRAY[
        'user_dashboard_layouts',
        'document_folders',
        'caldav_sync_versions',
        'caldav_change_log',
        'bexio_entity_mappings',
        'bexio_field_mappings',
        'bexio_sync_log',
        'lexware_entity_mappings',
        'lexware_field_mappings',
        'lexware_sync_log',
        'lexware_webhook_subscriptions',
        'datev_upload_configs',
        'datev_upload_log',
        'integration_channel_mappings',
        'integration_account_links',
        'integration_delivery_log',
        'chat_files',
        'message_reactions',
        'message_mentions',
        'call_sessions',
        'call_participants',
        'guest_sessions',
        'guest_channel_config',
        'integration_configs',
        'bexio_sync_configs',
        'lexware_sync_configs',
        'ticket_messages',
        'plugin_kv_store',
        'plugin_execution_log'
    ];
    _tbl     TEXT;
    _cnt     BIGINT;
    _report  TEXT := '';
BEGIN
    FOREACH _tbl IN ARRAY _tables LOOP
        EXECUTE format(
            'SELECT COUNT(*) FROM %I WHERE tenant_id IS NOT NULL
               AND tenant_id NOT IN (SELECT id FROM tenants)',
            _tbl
        ) INTO _cnt;
        IF _cnt > 0 THEN
            _report := _report || format('%s: %s orphaned row(s); ', _tbl, _cnt);
        END IF;
    END LOOP;

    IF _report <> '' THEN
        RAISE EXCEPTION
            'orphaned tenant_id rows block FK validation — fix before re-running: %',
            _report;
    END IF;
END;
$$;

-- ============================================================================
-- VALIDATE CONSTRAINTS
-- Source migration 000114 (2 constraints)
-- ============================================================================

ALTER TABLE user_dashboard_layouts VALIDATE CONSTRAINT fk_user_dashboard_layouts_tenant;
ALTER TABLE document_folders        VALIDATE CONSTRAINT fk_document_folders_tenant;

-- ============================================================================
-- Source migration 000115 (21 constraints)
-- ============================================================================

-- CalDAV Sync Layer
ALTER TABLE caldav_sync_versions         VALIDATE CONSTRAINT fk_caldav_sync_versions_tenant;
ALTER TABLE caldav_change_log            VALIDATE CONSTRAINT fk_caldav_change_log_tenant;

-- Integration Mappings (Bexio)
ALTER TABLE bexio_entity_mappings        VALIDATE CONSTRAINT fk_bexio_entity_mappings_tenant;
ALTER TABLE bexio_field_mappings         VALIDATE CONSTRAINT fk_bexio_field_mappings_tenant;
ALTER TABLE bexio_sync_log               VALIDATE CONSTRAINT fk_bexio_sync_log_tenant;

-- Integration Mappings (Lexware)
ALTER TABLE lexware_entity_mappings      VALIDATE CONSTRAINT fk_lexware_entity_mappings_tenant;
ALTER TABLE lexware_field_mappings       VALIDATE CONSTRAINT fk_lexware_field_mappings_tenant;
ALTER TABLE lexware_sync_log             VALIDATE CONSTRAINT fk_lexware_sync_log_tenant;
ALTER TABLE lexware_webhook_subscriptions VALIDATE CONSTRAINT fk_lexware_webhook_subscriptions_tenant;

-- Integration Mappings (DATEV)
ALTER TABLE datev_upload_configs         VALIDATE CONSTRAINT fk_datev_upload_configs_tenant;
ALTER TABLE datev_upload_log             VALIDATE CONSTRAINT fk_datev_upload_log_tenant;

-- Integration Generic
ALTER TABLE integration_channel_mappings VALIDATE CONSTRAINT fk_integration_channel_mappings_tenant;
ALTER TABLE integration_account_links    VALIDATE CONSTRAINT fk_integration_account_links_tenant;
ALTER TABLE integration_delivery_log     VALIDATE CONSTRAINT fk_integration_delivery_log_tenant;

-- Chat / Call / Presence
ALTER TABLE chat_files                   VALIDATE CONSTRAINT fk_chat_files_tenant;
ALTER TABLE message_reactions            VALIDATE CONSTRAINT fk_message_reactions_tenant;
ALTER TABLE message_mentions             VALIDATE CONSTRAINT fk_message_mentions_tenant;
ALTER TABLE call_sessions                VALIDATE CONSTRAINT fk_call_sessions_tenant;
ALTER TABLE call_participants            VALIDATE CONSTRAINT fk_call_participants_tenant;
ALTER TABLE guest_sessions               VALIDATE CONSTRAINT fk_guest_sessions_tenant;
ALTER TABLE guest_channel_config         VALIDATE CONSTRAINT fk_guest_channel_config_tenant;

-- ============================================================================
-- Source migration 000125 (3 constraints)
-- ============================================================================

ALTER TABLE integration_configs          VALIDATE CONSTRAINT fk_integration_configs_tenant;
ALTER TABLE bexio_sync_configs           VALIDATE CONSTRAINT fk_bexio_sync_configs_tenant;
ALTER TABLE lexware_sync_configs         VALIDATE CONSTRAINT fk_lexware_sync_configs_tenant;

-- ============================================================================
-- Source migration 000126 (3 constraints)
-- ============================================================================

ALTER TABLE ticket_messages              VALIDATE CONSTRAINT fk_ticket_messages_tenant;
ALTER TABLE plugin_kv_store              VALIDATE CONSTRAINT fk_plugin_kv_store_tenant;
ALTER TABLE plugin_execution_log         VALIDATE CONSTRAINT fk_plugin_execution_log_tenant;

-- Rollback: Option-B Phase 2 Batch 1 — Settings/Preferences/Document-Layer tenant_id

-- ============================================================================
-- GROUP D rollback
-- ============================================================================
DROP INDEX IF EXISTS idx_caldav_push_subscriptions_tenant_id;
ALTER TABLE caldav_push_subscriptions DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- GROUP C rollback
-- ============================================================================
DROP INDEX IF EXISTS idx_gdpr_deletion_requests_tenant_id;
ALTER TABLE gdpr_deletion_requests DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_recovery_codes_tenant_id;
ALTER TABLE recovery_codes DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_app_specific_passwords_tenant_id;
ALTER TABLE app_specific_passwords DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_user_sessions_tenant_user;
DROP INDEX IF EXISTS idx_user_sessions_tenant_id;
ALTER TABLE user_sessions DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- GROUP B rollback
-- ============================================================================
DROP INDEX IF EXISTS idx_wopi_locks_tenant_id;
ALTER TABLE wopi_locks DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_document_entity_links_tenant_id;
ALTER TABLE document_entity_links DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_document_file_tags_tenant_id;
ALTER TABLE document_file_tags DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_document_tags_tenant_id;
ALTER TABLE document_tags DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_document_shares_tenant_id;
ALTER TABLE document_shares DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_document_file_versions_tenant_id;
ALTER TABLE document_file_versions DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE document_folders DROP CONSTRAINT IF EXISTS fk_document_folders_tenant;
DROP INDEX IF EXISTS idx_document_folders_tenant_id;
ALTER TABLE document_folders DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- GROUP A rollback
-- ============================================================================
DROP INDEX IF EXISTS idx_task_custom_field_values_tenant_id;
ALTER TABLE task_custom_field_values DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_task_entity_links_tenant_id;
ALTER TABLE task_entity_links DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_user_project_preferences_tenant_id;
ALTER TABLE user_project_preferences DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE user_dashboard_layouts DROP CONSTRAINT IF EXISTS fk_user_dashboard_layouts_tenant;
DROP INDEX IF EXISTS idx_user_dashboard_layouts_tenant_id;
ALTER TABLE user_dashboard_layouts DROP COLUMN IF EXISTS tenant_id;

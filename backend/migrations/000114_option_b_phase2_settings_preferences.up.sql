-- Option-B Phase 2 Batch 1: add tenant_id to Settings, Dashboard, Document-Layer, Auth-Misc, CalDAV.
-- Sprint 3 S3.MT.2 — 2026-05-08
-- Sentinel: 00000000-0000-0000-0000-000000000001 (existing single-tenant data)
-- All ALTERs use ADD COLUMN IF NOT EXISTS to be idempotent.

-- ============================================================================
-- GROUP A: User Settings / Dashboard Preferences
-- ============================================================================

-- user_dashboard_layouts (user_id → users.tenant_id)
ALTER TABLE user_dashboard_layouts ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE user_dashboard_layouts udl
SET tenant_id = u.tenant_id
FROM users u
WHERE udl.user_id = u.id
  AND udl.tenant_id IS NULL;
ALTER TABLE user_dashboard_layouts ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE user_dashboard_layouts
    ADD CONSTRAINT fk_user_dashboard_layouts_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_user_dashboard_layouts_tenant_id ON user_dashboard_layouts(tenant_id);

-- user_project_preferences (user_id → users.tenant_id; PK is (user_id, project_id))
ALTER TABLE user_project_preferences ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE user_project_preferences upp
SET tenant_id = u.tenant_id
FROM users u
WHERE upp.user_id = u.id
  AND upp.tenant_id IS NULL;
ALTER TABLE user_project_preferences ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_project_preferences_tenant_id ON user_project_preferences(tenant_id);

-- task_entity_links (task_id → tasks.tenant_id)
ALTER TABLE task_entity_links ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE task_entity_links tel
SET tenant_id = t.tenant_id
FROM tasks t
WHERE tel.task_id = t.id
  AND tel.tenant_id IS NULL;
ALTER TABLE task_entity_links ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_task_entity_links_tenant_id ON task_entity_links(tenant_id);

-- task_custom_field_values (task_id → tasks.tenant_id; PK is (task_id, field_id))
ALTER TABLE task_custom_field_values ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE task_custom_field_values tcfv
SET tenant_id = t.tenant_id
FROM tasks t
WHERE tcfv.task_id = t.id
  AND tcfv.tenant_id IS NULL;
ALTER TABLE task_custom_field_values ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_task_custom_field_values_tenant_id ON task_custom_field_values(tenant_id);

-- ============================================================================
-- GROUP B: Document Layer
-- ============================================================================

-- document_folders (created_by → users.tenant_id; no direct tenant FK exists)
ALTER TABLE document_folders ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE document_folders df
SET tenant_id = u.tenant_id
FROM users u
WHERE df.created_by = u.id
  AND df.tenant_id IS NULL;
ALTER TABLE document_folders ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE document_folders
    ADD CONSTRAINT fk_document_folders_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_document_folders_tenant_id ON document_folders(tenant_id);

-- document_file_versions (file_id → document_files.tenant_id)
ALTER TABLE document_file_versions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE document_file_versions dfv
SET tenant_id = df.tenant_id
FROM document_files df
WHERE dfv.file_id = df.id
  AND dfv.tenant_id IS NULL;
ALTER TABLE document_file_versions ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_document_file_versions_tenant_id ON document_file_versions(tenant_id);

-- document_shares (shared_with_user_id → users.tenant_id)
ALTER TABLE document_shares ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE document_shares ds
SET tenant_id = u.tenant_id
FROM users u
WHERE ds.shared_with_user_id = u.id
  AND ds.tenant_id IS NULL;
-- NOTE: some shares may have no matching user if they were deleted; keep nullable for those rows.
-- Set NOT NULL only if all rows were backfilled (safe in single-tenant bootstrap).
ALTER TABLE document_shares ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_document_shares_tenant_id ON document_shares(tenant_id);

-- document_tags (standalone tag definitions; created_by → users.tenant_id)
ALTER TABLE document_tags ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE document_tags dt
SET tenant_id = u.tenant_id
FROM users u
WHERE dt.created_by = u.id
  AND dt.tenant_id IS NULL;
ALTER TABLE document_tags ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_document_tags_tenant_id ON document_tags(tenant_id);

-- document_file_tags (junction: file_id → document_files.tenant_id)
ALTER TABLE document_file_tags ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE document_file_tags dft
SET tenant_id = df.tenant_id
FROM document_files df
WHERE dft.file_id = df.id
  AND dft.tenant_id IS NULL;
ALTER TABLE document_file_tags ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_document_file_tags_tenant_id ON document_file_tags(tenant_id);

-- document_entity_links (file_id → document_files.tenant_id; file_id may be nullable)
-- Strategy: backfill via file_id where available; remaining rows get sentinel.
ALTER TABLE document_entity_links ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE document_entity_links del
SET tenant_id = df.tenant_id
FROM document_files df
WHERE del.file_id = df.id
  AND del.tenant_id IS NULL;
-- Rows without a valid file_id get the single-tenant sentinel
UPDATE document_entity_links
SET tenant_id = '00000000-0000-0000-0000-000000000001'
WHERE tenant_id IS NULL;
ALTER TABLE document_entity_links ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_document_entity_links_tenant_id ON document_entity_links(tenant_id);

-- wopi_locks (file_id → document_files.tenant_id; PK is file_id)
ALTER TABLE wopi_locks ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE wopi_locks wl
SET tenant_id = df.tenant_id
FROM document_files df
WHERE wl.file_id = df.id
  AND wl.tenant_id IS NULL;
ALTER TABLE wopi_locks ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_wopi_locks_tenant_id ON wopi_locks(tenant_id);

-- ============================================================================
-- GROUP C: Security/Auth-Misc
-- ============================================================================

-- user_sessions (user_id → users.tenant_id)
ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE user_sessions us
SET tenant_id = u.tenant_id
FROM users u
WHERE us.user_id = u.id
  AND us.tenant_id IS NULL;
ALTER TABLE user_sessions ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_sessions_tenant_id ON user_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_tenant_user ON user_sessions(tenant_id, user_id);

-- app_specific_passwords (user_id → users.tenant_id)
ALTER TABLE app_specific_passwords ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE app_specific_passwords asp
SET tenant_id = u.tenant_id
FROM users u
WHERE asp.user_id = u.id
  AND asp.tenant_id IS NULL;
ALTER TABLE app_specific_passwords ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_app_specific_passwords_tenant_id ON app_specific_passwords(tenant_id);

-- recovery_codes (user_id → users.tenant_id)
ALTER TABLE recovery_codes ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE recovery_codes rc
SET tenant_id = u.tenant_id
FROM users u
WHERE rc.user_id = u.id
  AND rc.tenant_id IS NULL;
ALTER TABLE recovery_codes ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_recovery_codes_tenant_id ON recovery_codes(tenant_id);

-- gdpr_deletion_requests (contact_id → contacts.tenant_id)
ALTER TABLE gdpr_deletion_requests ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE gdpr_deletion_requests gdr
SET tenant_id = c.tenant_id
FROM contacts c
WHERE gdr.contact_id = c.id
  AND gdr.tenant_id IS NULL;
-- Rows with orphaned contact_id get sentinel
UPDATE gdpr_deletion_requests
SET tenant_id = '00000000-0000-0000-0000-000000000001'
WHERE tenant_id IS NULL;
ALTER TABLE gdpr_deletion_requests ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_gdpr_deletion_requests_tenant_id ON gdpr_deletion_requests(tenant_id);

-- ============================================================================
-- GROUP D: CalDAV Layer
-- ============================================================================

-- caldav_push_subscriptions (user_id → users.tenant_id)
ALTER TABLE caldav_push_subscriptions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE caldav_push_subscriptions cps
SET tenant_id = u.tenant_id
FROM users u
WHERE cps.user_id = u.id
  AND cps.tenant_id IS NULL;
ALTER TABLE caldav_push_subscriptions ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_caldav_push_subscriptions_tenant_id ON caldav_push_subscriptions(tenant_id);

-- Option-B Phase 2 Batch 2: add tenant_id to Integration-Mappings, CalDAV-Sync, and Chat/Call tables.
-- Sprint 3 S3.MT.2 — 2026-05-08
-- Sentinel: 00000000-0000-0000-0000-000000000001 (existing single-tenant data)
-- All ALTERs use ADD COLUMN IF NOT EXISTS to be idempotent.

-- ============================================================================
-- GROUP D-REST: CalDAV Sync Layer
-- ============================================================================

-- caldav_sync_versions (PK is (collection_type, collection_id); no user FK)
-- collection_id refers to calendars.id or contacts groups — backfill via calendars first,
-- then the sentinel for any row not covered (contacts tables may differ).
ALTER TABLE caldav_sync_versions ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Attempt backfill via calendars (collection_type='calendar')
UPDATE caldav_sync_versions csv
SET tenant_id = c.tenant_id
FROM calendars c
WHERE csv.collection_id = c.id
  AND csv.tenant_id IS NULL;

-- Attempt backfill via contacts (collection_type='contact' or 'addressbook')
UPDATE caldav_sync_versions csv
SET tenant_id = ct.tenant_id
FROM contacts ct
WHERE csv.collection_id = ct.id
  AND csv.tenant_id IS NULL;

-- Remaining rows (orphaned or system collections) get sentinel
UPDATE caldav_sync_versions
SET tenant_id = '00000000-0000-0000-0000-000000000001'
WHERE tenant_id IS NULL;

ALTER TABLE caldav_sync_versions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE caldav_sync_versions
    ADD CONSTRAINT fk_caldav_sync_versions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_caldav_sync_versions_tenant_id ON caldav_sync_versions(tenant_id);

-- caldav_change_log (bigserial PK; same collection_id logic as caldav_sync_versions)
ALTER TABLE caldav_change_log ADD COLUMN IF NOT EXISTS tenant_id UUID;

UPDATE caldav_change_log ccl
SET tenant_id = c.tenant_id
FROM calendars c
WHERE ccl.collection_id = c.id
  AND ccl.tenant_id IS NULL;

UPDATE caldav_change_log ccl
SET tenant_id = ct.tenant_id
FROM contacts ct
WHERE ccl.collection_id = ct.id
  AND ccl.tenant_id IS NULL;

UPDATE caldav_change_log
SET tenant_id = '00000000-0000-0000-0000-000000000001'
WHERE tenant_id IS NULL;

ALTER TABLE caldav_change_log ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE caldav_change_log
    ADD CONSTRAINT fk_caldav_change_log_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_caldav_change_log_tenant_id ON caldav_change_log(tenant_id);

-- ============================================================================
-- GROUP E: Integration Mappings
-- ============================================================================

-- bexio_entity_mappings (config_id → integration_configs.tenant_id)
ALTER TABLE bexio_entity_mappings ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE bexio_entity_mappings bem
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE bem.config_id = ic.id
  AND bem.tenant_id IS NULL;
ALTER TABLE bexio_entity_mappings ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bexio_entity_mappings
    ADD CONSTRAINT fk_bexio_entity_mappings_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_bexio_entity_mappings_tenant_id ON bexio_entity_mappings(tenant_id);

-- bexio_field_mappings (config_id → integration_configs.tenant_id)
ALTER TABLE bexio_field_mappings ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE bexio_field_mappings bfm
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE bfm.config_id = ic.id
  AND bfm.tenant_id IS NULL;
ALTER TABLE bexio_field_mappings ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bexio_field_mappings
    ADD CONSTRAINT fk_bexio_field_mappings_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_bexio_field_mappings_tenant_id ON bexio_field_mappings(tenant_id);

-- bexio_sync_log (config_id → integration_configs.tenant_id)
ALTER TABLE bexio_sync_log ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE bexio_sync_log bsl
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE bsl.config_id = ic.id
  AND bsl.tenant_id IS NULL;
ALTER TABLE bexio_sync_log ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE bexio_sync_log
    ADD CONSTRAINT fk_bexio_sync_log_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_bexio_sync_log_tenant_id ON bexio_sync_log(tenant_id);

-- lexware_entity_mappings (config_id → integration_configs.tenant_id)
ALTER TABLE lexware_entity_mappings ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE lexware_entity_mappings lem
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE lem.config_id = ic.id
  AND lem.tenant_id IS NULL;
ALTER TABLE lexware_entity_mappings ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE lexware_entity_mappings
    ADD CONSTRAINT fk_lexware_entity_mappings_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_lexware_entity_mappings_tenant_id ON lexware_entity_mappings(tenant_id);

-- lexware_field_mappings (config_id → integration_configs.tenant_id)
ALTER TABLE lexware_field_mappings ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE lexware_field_mappings lfm
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE lfm.config_id = ic.id
  AND lfm.tenant_id IS NULL;
ALTER TABLE lexware_field_mappings ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE lexware_field_mappings
    ADD CONSTRAINT fk_lexware_field_mappings_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_lexware_field_mappings_tenant_id ON lexware_field_mappings(tenant_id);

-- lexware_sync_log (config_id → integration_configs.tenant_id)
ALTER TABLE lexware_sync_log ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE lexware_sync_log lsl
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE lsl.config_id = ic.id
  AND lsl.tenant_id IS NULL;
ALTER TABLE lexware_sync_log ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE lexware_sync_log
    ADD CONSTRAINT fk_lexware_sync_log_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_lexware_sync_log_tenant_id ON lexware_sync_log(tenant_id);

-- lexware_webhook_subscriptions (config_id → integration_configs.tenant_id)
ALTER TABLE lexware_webhook_subscriptions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE lexware_webhook_subscriptions lws
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE lws.config_id = ic.id
  AND lws.tenant_id IS NULL;
ALTER TABLE lexware_webhook_subscriptions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE lexware_webhook_subscriptions
    ADD CONSTRAINT fk_lexware_webhook_subscriptions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_lexware_webhook_subscriptions_tenant_id ON lexware_webhook_subscriptions(tenant_id);

-- datev_upload_configs (has config_id → integration_configs; backfill via that FK)
ALTER TABLE datev_upload_configs ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE datev_upload_configs duc
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE duc.config_id = ic.id
  AND duc.tenant_id IS NULL;
-- Orphaned rows get sentinel (datev configs not linked to an integration_config)
UPDATE datev_upload_configs
SET tenant_id = '00000000-0000-0000-0000-000000000001'
WHERE tenant_id IS NULL;
ALTER TABLE datev_upload_configs ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE datev_upload_configs
    ADD CONSTRAINT fk_datev_upload_configs_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_datev_upload_configs_tenant_id ON datev_upload_configs(tenant_id);

-- datev_upload_log (config_id → datev_upload_configs.tenant_id)
ALTER TABLE datev_upload_log ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE datev_upload_log dul
SET tenant_id = duc.tenant_id
FROM datev_upload_configs duc
WHERE dul.config_id = duc.id
  AND dul.tenant_id IS NULL;
ALTER TABLE datev_upload_log ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE datev_upload_log
    ADD CONSTRAINT fk_datev_upload_log_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_datev_upload_log_tenant_id ON datev_upload_log(tenant_id);

-- integration_channel_mappings (config_id → integration_configs.tenant_id)
ALTER TABLE integration_channel_mappings ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE integration_channel_mappings icm
SET tenant_id = ic.tenant_id
FROM integration_configs ic
WHERE icm.config_id = ic.id
  AND icm.tenant_id IS NULL;
ALTER TABLE integration_channel_mappings ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE integration_channel_mappings
    ADD CONSTRAINT fk_integration_channel_mappings_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_integration_channel_mappings_tenant_id ON integration_channel_mappings(tenant_id);

-- integration_account_links (kmuhub_user_id → users.tenant_id)
ALTER TABLE integration_account_links ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE integration_account_links ial
SET tenant_id = u.tenant_id
FROM users u
WHERE ial.kmuhub_user_id = u.id
  AND ial.tenant_id IS NULL;
ALTER TABLE integration_account_links ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE integration_account_links
    ADD CONSTRAINT fk_integration_account_links_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_integration_account_links_tenant_id ON integration_account_links(tenant_id);

-- integration_delivery_log (mapping_id → integration_channel_mappings.tenant_id)
ALTER TABLE integration_delivery_log ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE integration_delivery_log idl
SET tenant_id = icm.tenant_id
FROM integration_channel_mappings icm
WHERE idl.mapping_id = icm.id
  AND idl.tenant_id IS NULL;
ALTER TABLE integration_delivery_log ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE integration_delivery_log
    ADD CONSTRAINT fk_integration_delivery_log_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_integration_delivery_log_tenant_id ON integration_delivery_log(tenant_id);

-- integration_link_tokens (no direct tenant FK; ephemeral tokens — sentinel safe here)
-- These are short-lived and not queried by tenant; add tenant_id for completeness only.
ALTER TABLE integration_link_tokens ADD COLUMN IF NOT EXISTS tenant_id UUID;
-- No FK-based backfill possible (only platform+external_user_id); use sentinel.
UPDATE integration_link_tokens
SET tenant_id = '00000000-0000-0000-0000-000000000001'
WHERE tenant_id IS NULL;
ALTER TABLE integration_link_tokens ALTER COLUMN tenant_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_integration_link_tokens_tenant_id ON integration_link_tokens(tenant_id);

-- ============================================================================
-- GROUP F: Chat / Presence / Call
-- ============================================================================

-- chat_files (channel_id → channels.tenant_id)
ALTER TABLE chat_files ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE chat_files cf
SET tenant_id = ch.tenant_id
FROM channels ch
WHERE cf.channel_id = ch.id
  AND cf.tenant_id IS NULL;
ALTER TABLE chat_files ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE chat_files
    ADD CONSTRAINT fk_chat_files_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_chat_files_tenant_id ON chat_files(tenant_id);

-- message_reactions (message_id → messages.tenant_id)
ALTER TABLE message_reactions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE message_reactions mr
SET tenant_id = m.tenant_id
FROM messages m
WHERE mr.message_id = m.id
  AND mr.tenant_id IS NULL;
ALTER TABLE message_reactions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE message_reactions
    ADD CONSTRAINT fk_message_reactions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_message_reactions_tenant_id ON message_reactions(tenant_id);

-- message_mentions (message_id → messages.tenant_id)
ALTER TABLE message_mentions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE message_mentions mm
SET tenant_id = m.tenant_id
FROM messages m
WHERE mm.message_id = m.id
  AND mm.tenant_id IS NULL;
ALTER TABLE message_mentions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE message_mentions
    ADD CONSTRAINT fk_message_mentions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_message_mentions_tenant_id ON message_mentions(tenant_id);

-- call_sessions (initiator_id → users.tenant_id)
-- Must run before call_participants so the backfill path is available.
ALTER TABLE call_sessions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE call_sessions cs
SET tenant_id = u.tenant_id
FROM users u
WHERE cs.initiator_id = u.id
  AND cs.tenant_id IS NULL;
ALTER TABLE call_sessions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE call_sessions
    ADD CONSTRAINT fk_call_sessions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_call_sessions_tenant_id ON call_sessions(tenant_id);

-- call_participants (call_id → call_sessions.tenant_id; AFTER call_sessions above)
ALTER TABLE call_participants ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE call_participants cp
SET tenant_id = cs.tenant_id
FROM call_sessions cs
WHERE cp.call_id = cs.id
  AND cp.tenant_id IS NULL;
ALTER TABLE call_participants ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE call_participants
    ADD CONSTRAINT fk_call_participants_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_call_participants_tenant_id ON call_participants(tenant_id);

-- guest_sessions (channel_id → channels.tenant_id)
ALTER TABLE guest_sessions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE guest_sessions gs
SET tenant_id = ch.tenant_id
FROM channels ch
WHERE gs.channel_id = ch.id
  AND gs.tenant_id IS NULL;
ALTER TABLE guest_sessions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE guest_sessions
    ADD CONSTRAINT fk_guest_sessions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_guest_sessions_tenant_id ON guest_sessions(tenant_id);

-- guest_channel_config (channel_id → channels.tenant_id)
ALTER TABLE guest_channel_config ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE guest_channel_config gcc
SET tenant_id = ch.tenant_id
FROM channels ch
WHERE gcc.channel_id = ch.id
  AND gcc.tenant_id IS NULL;
ALTER TABLE guest_channel_config ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE guest_channel_config
    ADD CONSTRAINT fk_guest_channel_config_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_guest_channel_config_tenant_id ON guest_channel_config(tenant_id);

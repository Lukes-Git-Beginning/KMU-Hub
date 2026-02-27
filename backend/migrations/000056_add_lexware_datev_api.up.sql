-- Extend integration_configs platform CHECK to include 'lexware' and 'datev_api'
ALTER TABLE integration_configs DROP CONSTRAINT IF EXISTS integration_configs_platform_check;
ALTER TABLE integration_configs ADD CONSTRAINT integration_configs_platform_check
    CHECK (platform IN ('teams', 'slack', 'bexio', 'lexware', 'datev_api'));

-- ============================================================================
-- Lexware Office Tables
-- ============================================================================

-- Per-tenant Lexware sync configuration
CREATE TABLE lexware_sync_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    contact_sync_enabled BOOLEAN NOT NULL DEFAULT true,
    contact_sync_interval_minutes INTEGER NOT NULL DEFAULT 15,
    invoice_push_enabled BOOLEAN NOT NULL DEFAULT true,
    quote_push_enabled BOOLEAN NOT NULL DEFAULT true,
    webhook_enabled BOOLEAN NOT NULL DEFAULT true,
    last_contact_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(config_id)
);

-- KMU Hub ID <-> Lexware ID entity mappings
-- Note: Lexware uses UUID strings (VARCHAR 36), not integers like Bexio
CREATE TABLE lexware_entity_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('contact', 'invoice', 'quote', 'credit_note')),
    kmuhub_id UUID NOT NULL,
    lexware_id VARCHAR(36) NOT NULL,
    lexware_version INTEGER NOT NULL DEFAULT 0,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lexware_updated_at TIMESTAMPTZ,
    kmuhub_updated_at TIMESTAMPTZ,
    sync_direction VARCHAR(10) NOT NULL DEFAULT 'both' CHECK (sync_direction IN ('inbound', 'outbound', 'both')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_lexware_entity_mappings_kmuhub ON lexware_entity_mappings(config_id, entity_type, kmuhub_id);
CREATE UNIQUE INDEX idx_lexware_entity_mappings_lexware ON lexware_entity_mappings(config_id, entity_type, lexware_id);

-- Configurable field mapping per entity type
CREATE TABLE lexware_field_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('contact', 'invoice', 'quote', 'credit_note')),
    mappings JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(config_id, entity_type)
);

-- Sync operation audit trail
CREATE TABLE lexware_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    sync_type VARCHAR(20) NOT NULL CHECK (sync_type IN ('contact_full', 'contact_delta', 'invoice_push', 'quote_push', 'credit_note_push', 'webhook_event')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'partial')),
    items_processed INTEGER NOT NULL DEFAULT 0,
    items_created INTEGER NOT NULL DEFAULT 0,
    items_updated INTEGER NOT NULL DEFAULT 0,
    items_failed INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_lexware_sync_log_config ON lexware_sync_log(config_id, started_at DESC);

-- Lexware webhook subscriptions tracking
CREATE TABLE lexware_webhook_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    subscription_id VARCHAR(36) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    callback_url TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(config_id, event_type)
);

CREATE INDEX idx_lexware_webhook_subs_config ON lexware_webhook_subscriptions(config_id, is_active);

-- ============================================================================
-- DATEV Upload Tables
-- ============================================================================

-- Per-tenant DATEV API upload configuration
CREATE TABLE datev_upload_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    client_number VARCHAR(20) NOT NULL DEFAULT '',
    auto_upload_enabled BOOLEAN NOT NULL DEFAULT false,
    upload_after_export BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(config_id)
);

-- DATEV upload audit trail
CREATE TABLE datev_upload_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    upload_type VARCHAR(20) NOT NULL CHECK (upload_type IN ('buchungsstapel', 'belegbild')),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'uploading', 'completed', 'failed')),
    file_size INTEGER NOT NULL DEFAULT 0,
    document_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_datev_upload_log_config ON datev_upload_log(config_id, started_at DESC);

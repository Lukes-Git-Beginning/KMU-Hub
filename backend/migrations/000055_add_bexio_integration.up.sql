-- Extend integration_configs platform CHECK to include 'bexio'
ALTER TABLE integration_configs DROP CONSTRAINT IF EXISTS integration_configs_platform_check;
ALTER TABLE integration_configs ADD CONSTRAINT integration_configs_platform_check
    CHECK (platform IN ('teams', 'slack', 'bexio'));

-- Per-tenant Bexio sync configuration
CREATE TABLE bexio_sync_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    contact_sync_enabled BOOLEAN NOT NULL DEFAULT true,
    contact_sync_interval_minutes INTEGER NOT NULL DEFAULT 15,
    invoice_push_enabled BOOLEAN NOT NULL DEFAULT true,
    quote_push_enabled BOOLEAN NOT NULL DEFAULT true,
    payment_poll_enabled BOOLEAN NOT NULL DEFAULT true,
    payment_poll_interval_minutes INTEGER NOT NULL DEFAULT 5,
    last_contact_sync_at TIMESTAMPTZ,
    last_payment_poll_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(config_id)
);

-- KMU Hub ID <-> Bexio ID entity mappings
CREATE TABLE bexio_entity_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('contact', 'invoice', 'quote')),
    kmuhub_id UUID NOT NULL,
    bexio_id INTEGER NOT NULL,
    last_synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    bexio_updated_at TIMESTAMPTZ,
    kmuhub_updated_at TIMESTAMPTZ,
    sync_direction VARCHAR(10) NOT NULL DEFAULT 'both' CHECK (sync_direction IN ('inbound', 'outbound', 'both')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_bexio_entity_mappings_kmuhub ON bexio_entity_mappings(config_id, entity_type, kmuhub_id);
CREATE UNIQUE INDEX idx_bexio_entity_mappings_bexio ON bexio_entity_mappings(config_id, entity_type, bexio_id);

-- Configurable field mapping per entity type
CREATE TABLE bexio_field_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('contact', 'invoice', 'quote')),
    mappings JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(config_id, entity_type)
);

-- Sync operation audit trail
CREATE TABLE bexio_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    sync_type VARCHAR(20) NOT NULL CHECK (sync_type IN ('contact_full', 'contact_delta', 'invoice_push', 'quote_push', 'payment_poll')),
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

CREATE INDEX idx_bexio_sync_log_config ON bexio_sync_log(config_id, started_at DESC);

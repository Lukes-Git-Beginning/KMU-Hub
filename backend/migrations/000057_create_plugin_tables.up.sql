-- Plugin System Tables (Phase 20)

-- Plugin manifests: plugin definitions with metadata and configuration
CREATE TABLE plugin_manifests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version VARCHAR(50) NOT NULL DEFAULT '0.1.0',
    author VARCHAR(255) NOT NULL DEFAULT '',
    settings_schema JSONB NOT NULL DEFAULT '{}',
    permissions TEXT[] NOT NULL DEFAULT '{}',
    hook_registrations JSONB NOT NULL DEFAULT '[]',
    wasm_binary BYTEA,
    wasm_binary_hash VARCHAR(64),
    plugin_type VARCHAR(20) NOT NULL DEFAULT 'config' CHECK (plugin_type IN ('config', 'wasm')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plugin_manifests_slug ON plugin_manifests(slug);
CREATE INDEX idx_plugin_manifests_type ON plugin_manifests(plugin_type);

-- Plugin installations: per-tenant plugin installations
CREATE TABLE plugin_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    manifest_id UUID NOT NULL REFERENCES plugin_manifests(id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL DEFAULT 'pending_approval' CHECK (status IN ('pending_approval', 'active', 'disabled', 'error', 'uninstalled')),
    settings JSONB NOT NULL DEFAULT '{}',
    error_message TEXT,
    installed_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, manifest_id)
);

CREATE INDEX idx_plugin_installations_tenant ON plugin_installations(tenant_id);
CREATE INDEX idx_plugin_installations_status ON plugin_installations(tenant_id, status);
CREATE INDEX idx_plugin_installations_manifest ON plugin_installations(manifest_id);

-- Plugin permissions: approved permissions per installation
CREATE TABLE plugin_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id UUID NOT NULL REFERENCES plugin_installations(id) ON DELETE CASCADE,
    permission VARCHAR(100) NOT NULL,
    granted_by UUID NOT NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(installation_id, permission)
);

CREATE INDEX idx_plugin_permissions_installation ON plugin_permissions(installation_id);

-- Plugin KV store: plugin-scoped key-value storage
CREATE TABLE plugin_kv_store (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id UUID NOT NULL REFERENCES plugin_installations(id) ON DELETE CASCADE,
    key VARCHAR(500) NOT NULL,
    value JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(installation_id, key)
);

CREATE INDEX idx_plugin_kv_store_installation ON plugin_kv_store(installation_id);
CREATE INDEX idx_plugin_kv_store_key ON plugin_kv_store(installation_id, key);

-- Plugin execution log: audit trail for hook executions
CREATE TABLE plugin_execution_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_id UUID NOT NULL REFERENCES plugin_installations(id) ON DELETE CASCADE,
    hook_type VARCHAR(100) NOT NULL,
    module VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'error', 'timeout', 'skipped')),
    error_message TEXT,
    request_data JSONB,
    response_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plugin_execution_log_installation ON plugin_execution_log(installation_id, created_at DESC);
CREATE INDEX idx_plugin_execution_log_hook ON plugin_execution_log(hook_type, module);
CREATE INDEX idx_plugin_execution_log_status ON plugin_execution_log(status, created_at DESC);

-- Validation rules: config-based field validation
CREATE TABLE validation_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    installation_id UUID REFERENCES plugin_installations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    entity_type VARCHAR(50) NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    rule_type VARCHAR(30) NOT NULL CHECK (rule_type IN ('regex', 'range', 'required_if', 'format', 'custom', 'enum')),
    rule_config JSONB NOT NULL DEFAULT '{}',
    error_message VARCHAR(500) NOT NULL DEFAULT 'Validation failed',
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_validation_rules_tenant ON validation_rules(tenant_id);
CREATE INDEX idx_validation_rules_entity ON validation_rules(tenant_id, entity_type, enabled);
CREATE INDEX idx_validation_rules_field ON validation_rules(entity_type, field_name);

-- Workflow rules: config-based automation triggers
CREATE TABLE workflow_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    installation_id UUID REFERENCES plugin_installations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    trigger_event VARCHAR(100) NOT NULL,
    conditions JSONB NOT NULL DEFAULT '[]',
    actions JSONB NOT NULL DEFAULT '[]',
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflow_rules_tenant ON workflow_rules(tenant_id);
CREATE INDEX idx_workflow_rules_trigger ON workflow_rules(tenant_id, trigger_event, enabled);

-- Industry templates: preconfigured industry configurations
CREATE TABLE industry_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    industry VARCHAR(100) NOT NULL,
    icon VARCHAR(50) NOT NULL DEFAULT 'building',
    custom_fields JSONB NOT NULL DEFAULT '[]',
    validation_rules JSONB NOT NULL DEFAULT '[]',
    workflow_rules JSONB NOT NULL DEFAULT '[]',
    default_settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_industry_templates_slug ON industry_templates(slug);
CREATE INDEX idx_industry_templates_industry ON industry_templates(industry);

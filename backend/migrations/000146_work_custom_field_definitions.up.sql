-- Migration 000146: work_custom_field_definitions
-- Tenant-scoped custom field definitions for tasks.
-- Each definition specifies field_type and optional select options.

BEGIN;

CREATE TABLE work_custom_field_definitions (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    field_type VARCHAR(50)  NOT NULL
                            CHECK (field_type IN (
                                'text', 'number', 'date', 'boolean',
                                'select', 'multi_select', 'url', 'email', 'phone'
                            )),
    options    JSONB        NOT NULL DEFAULT '[]'::jsonb,
    position   INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX idx_work_custom_field_defs_tenant_pos
    ON work_custom_field_definitions (tenant_id, position);

-- RLS: standard tenant isolation
ALTER TABLE work_custom_field_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE work_custom_field_definitions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON work_custom_field_definitions
    USING  (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;

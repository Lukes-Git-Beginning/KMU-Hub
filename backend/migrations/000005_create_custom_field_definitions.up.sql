-- Custom field definitions for all CRM entities
-- Supports: text, number, date, boolean, select (single), multiselect

CREATE TABLE custom_field_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL,  -- 'contact', 'company', 'deal', 'activity'
    field_name VARCHAR(100) NOT NULL,
    field_label VARCHAR(200) NOT NULL,
    field_type VARCHAR(20) NOT NULL,   -- 'text', 'number', 'date', 'boolean', 'select', 'multiselect'
    options JSONB DEFAULT NULL,         -- For select/multiselect: ["option1", "option2"]
    default_value TEXT DEFAULT NULL,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_field_type CHECK (field_type IN ('text', 'number', 'date', 'boolean', 'select', 'multiselect')),
    CONSTRAINT valid_entity_type CHECK (entity_type IN ('contact', 'company', 'deal', 'activity'))
);

-- Unique constraint: one field_name per entity_type
CREATE UNIQUE INDEX idx_custom_field_definitions_entity_name
    ON custom_field_definitions (entity_type, field_name);

-- Index for listing fields by entity type
CREATE INDEX idx_custom_field_definitions_entity_type
    ON custom_field_definitions (entity_type, sort_order);

-- Add CRM-related permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('custom_fields:read', 'custom_fields', 'read'),
    ('custom_fields:write', 'custom_fields', 'write'),
    ('custom_fields:delete', 'custom_fields', 'delete');

-- Assign custom_fields permissions to admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'custom_fields';

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'custom_fields' AND p.action IN ('read', 'write');

-- Assign read to member
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'custom_fields' AND p.action = 'read';

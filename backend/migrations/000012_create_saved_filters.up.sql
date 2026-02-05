-- Saved Filters table for CRM
-- Allows users to save and reuse filter configurations

CREATE TABLE saved_filters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,  -- contact, company, deal, activity
    filter_json JSONB NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_saved_filters_created_by ON saved_filters (created_by);
CREATE INDEX idx_saved_filters_entity_type ON saved_filters (entity_type);
-- Only one default filter per user per entity type
CREATE UNIQUE INDEX idx_saved_filters_unique_default
    ON saved_filters (created_by, entity_type) WHERE is_default = TRUE;

-- Permissions (idempotent)
INSERT INTO permissions (name, resource, action) VALUES
    ('saved_filters:read', 'saved_filters', 'read'),
    ('saved_filters:write', 'saved_filters', 'write'),
    ('saved_filters:delete', 'saved_filters', 'delete'),
    ('reports:read', 'reports', 'read')
ON CONFLICT (name) DO NOTHING;

-- Assign saved_filters permissions to admin (all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'saved_filters'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign reports:read to admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.name = 'reports:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.resource = 'saved_filters' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign reports:read to manager
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager' AND p.name = 'reports:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign read + write to member (members can manage their own filters)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.resource = 'saved_filters' AND p.action IN ('read', 'write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Assign reports:read to member
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member' AND p.name = 'reports:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Migration 000178: hr_time_categories table

CREATE TABLE IF NOT EXISTS hr_time_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    color VARCHAR(7) NOT NULL DEFAULT '#6b7280',
    icon VARCHAR(50) NOT NULL DEFAULT 'Tag',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CALL enable_tenant_rls('hr_time_categories');

INSERT INTO permissions (name, resource, action) VALUES
    ('hr:time_category:read',  'hr:time_category', 'read'),
    ('hr:time_category:write', 'hr:time_category', 'write')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'hr:time_category'
ON CONFLICT (role_id, permission_id) DO NOTHING;

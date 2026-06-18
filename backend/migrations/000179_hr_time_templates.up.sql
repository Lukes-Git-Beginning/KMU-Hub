-- Migration 000179: hr_time_templates table

CREATE TABLE IF NOT EXISTS hr_time_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    category_id UUID NULL REFERENCES hr_time_categories(id) ON DELETE SET NULL,
    description TEXT NOT NULL DEFAULT '',
    estimated_minutes INT NOT NULL DEFAULT 30,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CALL enable_tenant_rls('hr_time_templates');

INSERT INTO permissions (name, resource, action) VALUES
    ('hr:time_template:read',  'hr:time_template', 'read'),
    ('hr:time_template:write', 'hr:time_template', 'write')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'hr:time_template'
ON CONFLICT (role_id, permission_id) DO NOTHING;

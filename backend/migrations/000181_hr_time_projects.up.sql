-- Migration 000181: hr_time_projects table

CREATE TABLE IF NOT EXISTS hr_time_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    customer_name TEXT NOT NULL DEFAULT '',
    color VARCHAR(7) NOT NULL DEFAULT '#6366f1',
    billable_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CALL enable_tenant_rls('hr_time_projects');

INSERT INTO permissions (name, resource, action) VALUES
    ('hr:time_project:read',  'hr:time_project', 'read'),
    ('hr:time_project:write', 'hr:time_project', 'write')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'hr:time_project'
ON CONFLICT (role_id, permission_id) DO NOTHING;

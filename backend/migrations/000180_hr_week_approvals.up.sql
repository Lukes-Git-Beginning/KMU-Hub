-- Migration 000180: hr_week_approvals table

CREATE TABLE IF NOT EXISTS hr_week_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    employee_id UUID NOT NULL,
    week_start DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open','submitted','approved','rejected')),
    submitted_at TIMESTAMPTZ NULL,
    approved_by UUID NULL,
    approved_at TIMESTAMPTZ NULL,
    rejection_reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, employee_id, week_start)
);

CALL enable_tenant_rls('hr_week_approvals');

INSERT INTO permissions (name, resource, action) VALUES
    ('hr:week_approval:read',  'hr:week_approval', 'read'),
    ('hr:week_approval:write', 'hr:week_approval', 'write')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin' AND p.resource = 'hr:week_approval'
ON CONFLICT (role_id, permission_id) DO NOTHING;

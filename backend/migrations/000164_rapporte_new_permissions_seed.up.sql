-- Migration 000164: RBAC seed for rapporte measurement + template permissions.
-- Follows pattern of 000093_seed_rapporte_permissions.
-- Roles that already have rapporte:report:read/write (admin) receive these too.

INSERT INTO permissions (name, resource, action) VALUES
    ('rapporte:measurement:read',  'rapporte:measurement', 'read'),
    ('rapporte:measurement:write', 'rapporte:measurement', 'write'),
    ('rapporte:template:read',     'rapporte:template',    'read'),
    ('rapporte:template:write',    'rapporte:template',    'write')
ON CONFLICT (name) DO NOTHING;

-- Grant to admin (same roles that have rapporte:report:read/write)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('rapporte:measurement', 'rapporte:template')
ON CONFLICT (role_id, permission_id) DO NOTHING;

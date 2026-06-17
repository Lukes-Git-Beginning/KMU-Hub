-- Migration 000161: seed permissions for shift swap requests

INSERT INTO permissions (name, resource, action) VALUES
    ('schichten:swap:create',  'schichten:swap', 'create'),
    ('schichten:swap:read',    'schichten:swap', 'read'),
    ('schichten:swap:approve', 'schichten:swap', 'approve')
ON CONFLICT (name) DO NOTHING;

-- Grant all schichten:swap permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource = 'schichten:swap'
ON CONFLICT (role_id, permission_id) DO NOTHING;

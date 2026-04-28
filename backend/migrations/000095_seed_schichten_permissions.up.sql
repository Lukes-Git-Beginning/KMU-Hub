-- Sprint 2 W2.A Schichten — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('schichten:shift:read',      'schichten:shift',      'read'),
    ('schichten:shift:write',     'schichten:shift',      'write'),
    ('schichten:assignment:read', 'schichten:assignment', 'read'),
    ('schichten:assignment:write','schichten:assignment',  'write'),
    ('schichten:template:read',   'schichten:template',   'read'),
    ('schichten:template:write',  'schichten:template',   'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all schichten permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('schichten:shift', 'schichten:assignment', 'schichten:template')
ON CONFLICT (role_id, permission_id) DO NOTHING;

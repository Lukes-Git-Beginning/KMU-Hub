-- Sprint 2 W1.A Einkauf — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('einkauf:supplier:read',  'einkauf:supplier', 'read'),
    ('einkauf:supplier:write', 'einkauf:supplier', 'write'),
    ('einkauf:po:read',        'einkauf:po',       'read'),
    ('einkauf:po:write',       'einkauf:po',       'write'),
    ('einkauf:line:read',      'einkauf:line',     'read'),
    ('einkauf:line:write',     'einkauf:line',     'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all einkauf permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('einkauf:supplier', 'einkauf:po', 'einkauf:line')
ON CONFLICT (role_id, permission_id) DO NOTHING;

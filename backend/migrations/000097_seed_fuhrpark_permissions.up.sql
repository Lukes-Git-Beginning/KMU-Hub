-- Sprint 2 W2.A Fuhrpark — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('fuhrpark:vehicle:read',   'fuhrpark:vehicle',  'read'),
    ('fuhrpark:vehicle:write',  'fuhrpark:vehicle',  'write'),
    ('fuhrpark:service:read',   'fuhrpark:service',  'read'),
    ('fuhrpark:service:write',  'fuhrpark:service',  'write'),
    ('fuhrpark:damage:read',    'fuhrpark:damage',   'read'),
    ('fuhrpark:damage:write',   'fuhrpark:damage',   'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all fuhrpark permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('fuhrpark:vehicle', 'fuhrpark:service', 'fuhrpark:damage')
ON CONFLICT (role_id, permission_id) DO NOTHING;

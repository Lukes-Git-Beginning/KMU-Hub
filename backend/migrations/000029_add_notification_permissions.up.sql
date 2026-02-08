INSERT INTO permissions (name, resource, action) VALUES
    ('notifications:read', 'notifications', 'read'),
    ('notifications:write', 'notifications', 'write')
ON CONFLICT (name) DO NOTHING;

-- Grant notification permissions to all existing roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE p.resource = 'notifications'
ON CONFLICT (role_id, permission_id) DO NOTHING;

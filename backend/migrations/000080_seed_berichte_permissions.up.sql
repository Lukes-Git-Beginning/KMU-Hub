-- Sprint 1 S1.2 Berichte — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('berichte:reports:read',  'berichte:reports', 'read'),
    ('berichte:reports:write', 'berichte:reports', 'write')
ON CONFLICT (name) DO NOTHING;

-- Grant berichte:reports permissions to admin role (all existing admin roles)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource = 'berichte:reports'
ON CONFLICT (role_id, permission_id) DO NOTHING;

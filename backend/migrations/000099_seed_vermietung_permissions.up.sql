-- Sprint 2 W2.A Vermietung — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('vermietung:object:read',      'vermietung:object',     'read'),
    ('vermietung:object:write',     'vermietung:object',     'write'),
    ('vermietung:rental:read',      'vermietung:rental',     'read'),
    ('vermietung:rental:write',     'vermietung:rental',     'write'),
    ('vermietung:inspection:read',  'vermietung:inspection', 'read'),
    ('vermietung:inspection:write', 'vermietung:inspection', 'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all vermietung permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('vermietung:object', 'vermietung:rental', 'vermietung:inspection')
ON CONFLICT (role_id, permission_id) DO NOTHING;

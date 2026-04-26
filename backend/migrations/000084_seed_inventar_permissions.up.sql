-- Sprint 2 W1.A Inventar — ACL seed

INSERT INTO permissions (name, resource, action) VALUES
    ('inventar:item:read',     'inventar:item',     'read'),
    ('inventar:item:write',    'inventar:item',     'write'),
    ('inventar:movement:read', 'inventar:movement', 'read'),
    ('inventar:movement:write','inventar:movement', 'write'),
    ('inventar:warning:read',  'inventar:warning',  'read'),
    ('inventar:warning:write', 'inventar:warning',  'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all inventar permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('inventar:item', 'inventar:movement', 'inventar:warning')
ON CONFLICT (role_id, permission_id) DO NOTHING;

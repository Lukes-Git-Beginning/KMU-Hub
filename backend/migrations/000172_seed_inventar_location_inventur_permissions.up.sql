-- 000172 Permissions für inventory_locations und inventur_sessions

INSERT INTO permissions (name, resource, action) VALUES
    ('inventar:location:read',   'inventar:location',   'read'),
    ('inventar:location:write',  'inventar:location',   'write'),
    ('inventar:inventur:read',   'inventar:inventur',   'read'),
    ('inventar:inventur:write',  'inventar:inventur',   'write')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('inventar:location', 'inventar:inventur')
ON CONFLICT (role_id, permission_id) DO NOTHING;

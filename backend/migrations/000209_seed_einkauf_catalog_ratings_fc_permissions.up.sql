-- 000209 Permissions für Einkauf Catalog, Ratings und Framework Contracts

INSERT INTO permissions (name, resource, action) VALUES
    ('einkauf:catalog:read',   'einkauf:catalog',  'read'),
    ('einkauf:catalog:write',  'einkauf:catalog',  'write'),
    ('einkauf:rating:read',    'einkauf:rating',   'read'),
    ('einkauf:rating:write',   'einkauf:rating',   'write'),
    ('einkauf:contract:read',  'einkauf:contract', 'read'),
    ('einkauf:contract:write', 'einkauf:contract', 'write')
ON CONFLICT (name) DO NOTHING;

-- Grant all new einkauf permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.resource IN ('einkauf:catalog', 'einkauf:rating', 'einkauf:contract')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Seed module-grants:read/write for the user_module_grants endpoints (R3-E6).
-- Without these rows RequirePermission("module-grants", ...) returns 403 for
-- everyone including admins. Pattern follows 000131/000136/000138.

INSERT INTO permissions (name, resource, action) VALUES
    ('module-grants:read',  'module-grants', 'read'),
    ('module-grants:write', 'module-grants', 'write')
ON CONFLICT (name) DO NOTHING;

-- module-grants:read  — admin + manager (view who has which module access)
-- module-grants:write — admin only (grant/revoke module access)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN ('module-grants:read', 'module-grants:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.name = 'module-grants:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Migration 000136 — RBAC permission seed for booking-pages CRUD
-- Without these rows RequirePermission("booking-pages", ...) returns 403 for everyone,
-- including admins. Pattern follows 000129_seed_missing_module_permissions.

INSERT INTO permissions (name, resource, action) VALUES
    ('booking-pages:read',   'booking-pages', 'read'),
    ('booking-pages:write',  'booking-pages', 'write'),
    ('booking-pages:delete', 'booking-pages', 'delete')
ON CONFLICT (name) DO NOTHING;

-- Grant all three to the admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN ('booking-pages:read', 'booking-pages:write', 'booking-pages:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

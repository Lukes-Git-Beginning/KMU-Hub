-- Migration 000241 — RBAC seed for the inventar item-attachment routes
-- (GET/POST /api/v1/inventar/items/{id}/attachments, DELETE
-- /api/v1/inventar/attachments/{id}). Without these rows the new
-- RequirePermission("inventar:attachment", ...) guard returns 403 for everyone.
-- Granted to admin, matching the existing inventar permission seeds (000084/000185).
DO $$
DECLARE v_admin_role_id UUID;
BEGIN
    SELECT id INTO v_admin_role_id FROM roles WHERE name = 'admin' LIMIT 1;
    IF v_admin_role_id IS NOT NULL THEN
        INSERT INTO permissions (name, resource, action) VALUES
            ('inventar:attachment:read',  'inventar:attachment', 'read'),
            ('inventar:attachment:write', 'inventar:attachment', 'write')
        ON CONFLICT (name) DO NOTHING;
        INSERT INTO role_permissions (role_id, permission_id)
        SELECT v_admin_role_id, p.id FROM permissions p
        WHERE p.name IN ('inventar:attachment:read', 'inventar:attachment:write')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

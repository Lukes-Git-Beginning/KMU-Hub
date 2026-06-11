-- Migration 000144 — RBAC permission seed for the generic presigned-file routes
-- (POST /api/v1/files/presign-upload, GET /api/v1/files/presign-download).
-- Without these rows RequirePermission("files", ...) returns 403 for everyone,
-- including admins. Pattern follows 000131_seed_meetings_write_permission.

INSERT INTO permissions (name, resource, action) VALUES
    ('files:read',  'files', 'read'),
    ('files:write', 'files', 'write')
ON CONFLICT (name) DO NOTHING;

-- Grant to admin, manager AND member: presigned uploads back day-to-day flows
-- (profile avatar, chat attachments, report photos). Tenant isolation is
-- enforced server-side via the {tenant_id}/ object-key prefix, and the
-- per-module write permissions still guard attaching files to entities.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager', 'member')
  AND p.name IN ('files:read', 'files:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

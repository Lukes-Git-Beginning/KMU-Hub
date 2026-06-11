-- Migration 000147: RBAC permission seeds for work_labels and work_custom_fields.
-- Without these rows RequirePermission("work_labels", ...) and
-- RequirePermission("work_custom_fields", ...) return 403 for everyone,
-- including admins. Pattern follows 000144_seed_files_permissions.

-- work_labels permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('work_labels:read',   'work_labels', 'read'),
    ('work_labels:write',  'work_labels', 'write'),
    ('work_labels:delete', 'work_labels', 'delete')
ON CONFLICT (name) DO NOTHING;

-- work_custom_fields permissions
INSERT INTO permissions (name, resource, action) VALUES
    ('work_custom_fields:read',   'work_custom_fields', 'read'),
    ('work_custom_fields:write',  'work_custom_fields', 'write'),
    ('work_custom_fields:delete', 'work_custom_fields', 'delete')
ON CONFLICT (name) DO NOTHING;

-- read + write for admin, manager, member
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager', 'member')
  AND p.name IN (
      'work_labels:read',   'work_labels:write',
      'work_custom_fields:read', 'work_custom_fields:write'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- delete for admin and manager only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager')
  AND p.name IN ('work_labels:delete', 'work_custom_fields:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

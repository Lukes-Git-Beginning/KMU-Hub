-- Migration 000100: Add rapporte:report:approve permission (Bug #3)
-- Separate approve/reject from generic write to allow RBAC distinction

INSERT INTO permissions (name, resource, action)
VALUES
    ('rapporte:report:approve', 'rapporte:report', 'approve')
ON CONFLICT (name) DO NOTHING;

-- Grant approve permission to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name = 'rapporte:report:approve'
ON CONFLICT (role_id, permission_id) DO NOTHING;

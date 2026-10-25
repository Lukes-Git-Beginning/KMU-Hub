-- Restores the member grant of 000068.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name = 'member'
  AND p.name = 'dialer:agent:manage'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Revoke finance:read + finance:write from the manager role.
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'manager')
  AND permission_id IN (
    SELECT id FROM permissions WHERE name IN ('finance:read', 'finance:write')
  );

-- Removes the member grant of 000341.
DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name = 'member'
  AND p.name = 'hr:time_project:read';

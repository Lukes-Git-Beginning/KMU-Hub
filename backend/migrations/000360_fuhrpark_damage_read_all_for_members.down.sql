UPDATE role_permissions rp
SET scope = 'own'
FROM roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name = 'member'
  AND p.name = 'fuhrpark:damage:read';

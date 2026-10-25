-- Restore the manager preset's scope for the two absence keys to 'all' (migration 256).
UPDATE role_permissions rp
SET scope = 'all'
FROM roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name = 'manager'
  AND p.name IN ('team:absence:approve', 'team:absence:read');

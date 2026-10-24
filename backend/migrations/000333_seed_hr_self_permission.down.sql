-- Revoke the hr:self preset grants and the permission row added by the up migration.
DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name IN ('admin', 'manager', 'member')
  AND p.name = 'hr:self';

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'hr:self');

DELETE FROM permissions WHERE name = 'hr:self';

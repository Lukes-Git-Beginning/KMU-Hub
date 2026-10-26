DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id AND rp.permission_id = p.id
  AND r.name = 'member' AND r.tenant_id IS NULL
  AND p.name IN ('wiki:article:delete', 'wiki:share_token:create');

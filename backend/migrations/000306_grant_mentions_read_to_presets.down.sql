-- Revoke the mentions:read preset grants added by the up migration.
--
-- Scoped to the three system presets and to that one permission, so a grant a
-- tenant added by hand on a custom role is left alone.
DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name IN ('admin', 'manager', 'member')
  AND p.name = 'mentions:read';

-- Revoke the own-delete grants of the manager and member presets added by the up migration.
DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name IN ('manager', 'member')
  AND p.name IN ('calendars:delete', 'messages:delete', 'files:delete');

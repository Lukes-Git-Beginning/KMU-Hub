DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name IN ('manager', 'member')
  AND p.name IN ('fuhrpark:booking:read', 'fuhrpark:booking:write',
                 'fuhrpark:license:read', 'fuhrpark:license:write',
                 'fuhrpark:damage:read',
                 'produktion:workstep:read', 'vermietung:inspection:read');

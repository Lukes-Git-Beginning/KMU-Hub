-- search:read for readonly, it_admin and hr_admin, so the header search stops answering 403 to them.
--
-- Migration 338 gave the key to member and manager. These three presets hold crm:contact:read and
-- documents:file:read but not search:read, and the header search bar is not permission-gated. The
-- gateway asks each source only when the caller holds that source's own read key
-- (route_search_global.go), so the grant finds no row the role could not already open in its module.
-- extern stays without it on purpose: a guest must not get a tenant-wide document search.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('readonly', 'it_admin', 'hr_admin')
  AND p.name = 'search:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

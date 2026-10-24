-- search:read for member and manager, so the header search stops answering 403 to every employee.
--
-- Migration 129 gave search:read to admin only. The key opens GET /api/v1/search/global and nothing
-- else; the gateway asks each source only when the caller holds that source's own read key (CRM per
-- entity type, documents, email — route_search_global.go), so the grant finds no row the caller
-- could not already open in its module.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO permissions (name, resource, action) VALUES
    ('search:read', 'search', 'read')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('manager', 'member')
  AND p.name = 'search:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

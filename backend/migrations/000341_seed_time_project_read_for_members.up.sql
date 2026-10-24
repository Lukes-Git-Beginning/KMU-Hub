-- hr:time_project:read for member.
--
-- The manual time entry dialog (profile > time tracking) fills its project picker from
-- GET /hr/time/projects. Only admin held the key (migration 000181; manager has it since 000224), so
-- a member saw an empty picker on their own entry. Projects are master data (name, customer
-- name, colour, billable), the same tenant-wide catalogue manager already reads: scope 'all'.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name = 'member'
  AND p.name = 'hr:time_project:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

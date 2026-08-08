-- Grant mentions:read to the presets that already hold messages:read.
--
-- GET /api/v1/messages/mentions has been guarded by RequirePermission("mentions",
-- "read") since the route was added, and migration 000017 created the permission
-- row -- but no migration ever granted it to a role. The permission existed while
-- belonging to nobody, which is exactly the lockout shape this repo has hit
-- before: the route answered 403 for every user, admin included.
--
-- Verified 2026-08-08 against the local database at migration 305: mentions:read
-- was the ONLY permission in the table with no system-role grant at all.
--
-- The three presets mirror messages:read exactly (admin, manager, member at scope
-- 'all'). The mentions route sits in the same /api/v1/messages block and serves
-- the same audience; readonly, it_admin and hr_admin hold no messages:read either
-- and stay unchanged rather than gaining a right by accident.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('admin', 'manager', 'member')
  AND p.name = 'mentions:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

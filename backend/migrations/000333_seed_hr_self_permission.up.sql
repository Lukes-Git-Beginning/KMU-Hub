-- hr:self: the self-service key of the HR module.
--
-- hr:read and hr:write were seeded for admin only (migration 129), so every
-- manager and member answered 403 on leave requests, the time clock, their own
-- entries and the weekly submission. hr:read/hr:write also cover the
-- supervisory side (foreign balances, sick leave, employee files), which an
-- employee must not get. hr:self carries only the routes that act on the
-- caller's own identity; the gateway guards accept it additively beside the
-- coarse keys.
--
-- Decision Luke 2026-10-24: member and manager at scope 'own', admin at 'all'.
INSERT INTO permissions (name, resource, action) VALUES
    ('hr:self', 'hr', 'self')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('manager', 'member')
  AND p.name = 'hr:self'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name = 'admin'
  AND p.name = 'hr:self'
ON CONFLICT (role_id, permission_id) DO NOTHING;

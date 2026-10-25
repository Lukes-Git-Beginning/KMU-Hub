-- email:account:write|delete for member and manager at scope 'own', admin at 'all' (decision
-- 2026-10-25: members set up their own mailbox; user_id pinned, change/remove only their own).
--
-- Account create, update, delete and the connection test required the coarse email:write|delete,
-- which only admin holds (000129, 000337), because they dial any host. Since the dial guard
-- (security/safedial) refuses internal addresses, the gateway accepts these keys additively. At
-- scope 'own' the gateway pins user_id to the caller on create, and the email server confines
-- update and delete to the caller's own accounts unless the coarse key crosses the hop. Admin keeps
-- provisioning accounts for others through the coarse keys.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO permissions (name, resource, action) VALUES
    ('email:account:write', 'email:account', 'write'),
    ('email:account:delete', 'email:account', 'delete')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name IN ('member', 'manager') AND r.tenant_id IS NULL
  AND p.name IN ('email:account:write', 'email:account:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.name = 'admin' AND r.tenant_id IS NULL
  AND p.name IN ('email:account:write', 'email:account:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

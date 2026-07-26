-- Revert 000251. Dropping the role takes its user_roles rows with it
-- (ON DELETE CASCADE), which is the point: without the permission the role is
-- an empty label, and leaving it assigned would suggest a capability nobody has.
--
-- Tenants provisioned while the role existed are untouched. Rolling back the
-- ability to create tenants does not mean unmaking the ones that exist.

BEGIN;

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE name = 'tenants:write');

DELETE FROM permissions WHERE name = 'tenants:write';
DELETE FROM roles WHERE name = 'platform_admin';

COMMIT;

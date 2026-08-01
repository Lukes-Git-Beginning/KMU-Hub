-- Reverse of 000256.
--
-- Two deliberate asymmetries with the up migration:
--
-- 1) The `permissions` rows and the grants seeded onto admin/manager/member are
--    NOT deleted. The seed ran with ON CONFLICT, so a large part of those rows
--    predates this migration (earlier per-module permission seeds) and nothing
--    in the row distinguishes the two. Deleting them would revoke access that
--    other migrations granted. Extra catalogue rows without a grant are inert.
--    The four preset roles this migration created ARE removed — they are
--    unambiguous, and the cascade takes their grants with them.
--
-- 2) Restoring the globally-unique idx_roles_name fails if two tenants created
--    a custom role of the same name in the meantime. That is intended: silently
--    dropping one of them would delete a tenant's role assignment.

SET LOCAL row_security = off;

BEGIN;

DELETE FROM roles WHERE tenant_id IS NULL AND name IN ('it_admin', 'hr_admin', 'readonly', 'extern');

DROP POLICY IF EXISTS tenant_isolation_read ON role_permissions;
DROP POLICY IF EXISTS tenant_isolation_write ON role_permissions;
ALTER TABLE role_permissions DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_read ON roles;
DROP POLICY IF EXISTS tenant_isolation_write ON roles;
ALTER TABLE roles DISABLE ROW LEVEL SECURITY;

ALTER TABLE role_permissions DROP COLUMN scope;

DROP INDEX idx_roles_tenant_name;

ALTER TABLE roles
    DROP COLUMN color,
    DROP COLUMN based_on,
    DROP COLUMN tenant_id;

CREATE UNIQUE INDEX idx_roles_name ON roles (name);

COMMIT;

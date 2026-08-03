-- Backlog unit g-user-roles-rls (Block B / RLS gap audit): user_roles is the
-- last table of the RBAC core without tenant_id or a policy of its own. Every
-- read path so far has stayed safe only because it joins through users/roles
-- (both RLS-protected) by hand — this migration gives the table its own
-- backstop so a query that forgets that join fails closed instead of leaking
-- across tenants.
--
-- RLS alone does not fix the second, sharper problem this table has: the
-- legacy name-based AssignRole/RemoveRole run under sysctx.With() at
-- registration (Register assigns "member" to every new account), where RLS
-- does not filter anything. Migration 000256 let a tenant name a custom role
-- identically to a preset ("member", "admin", ...) — the coexisting row does
-- not violate idx_roles_tenant_name because presets use tenant_id IS NULL and
-- a custom role uses a real tenant_id. `SELECT r.id FROM roles r WHERE
-- r.name = $2` then matches BOTH rows, and AssignRole's INSERT ... SELECT
-- silently assigns every new registrant, of every tenant, into that
-- tenant's custom role too. That half of the fix lives in Go
-- (postgres_repository.go: AssignRole/RemoveRole now resolve system presets
-- only, tenant_id IS NULL), not in this migration — this migration only adds
-- the backstop column and policy so a future write path fails the same way
-- the refresh_tokens/plugin_permissions gap did (WITH CHECK, not a silent
-- cross-tenant row).

ALTER TABLE user_roles ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

UPDATE user_roles ur
SET tenant_id = u.tenant_id
FROM users u
WHERE u.id = ur.user_id;

ALTER TABLE user_roles ALTER COLUMN tenant_id SET NOT NULL;

CREATE INDEX idx_user_roles_tenant_id ON user_roles(tenant_id);

-- No preset-visibility split like roles/role_permissions: a user_roles row is
-- always an ASSIGNMENT, owned by the account's own tenant, never a
-- system-global row (the role it points at may be a preset, the assignment
-- itself never is). The plain single-policy helper is correct here.
CALL enable_tenant_rls('user_roles');

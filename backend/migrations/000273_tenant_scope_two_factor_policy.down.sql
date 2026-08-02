-- Reverse of 000273. Going back to one globally unique row per role is lossy by
-- definition: if two tenants have diverged, only one of their policies can
-- survive. The most recently edited row wins, which at least keeps the newest
-- administrator decision rather than an arbitrary one.

DROP POLICY IF EXISTS tenant_isolation ON two_factor_policy;
ALTER TABLE two_factor_policy NO FORCE ROW LEVEL SECURITY;
ALTER TABLE two_factor_policy DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_two_factor_policy_tenant_role;

DELETE FROM two_factor_policy
 WHERE id NOT IN (
     SELECT DISTINCT ON (role_name) id
     FROM two_factor_policy
     ORDER BY role_name, updated_at DESC, id
 );

ALTER TABLE two_factor_policy DROP COLUMN IF EXISTS tenant_id;

CREATE UNIQUE INDEX idx_two_factor_policy_role ON two_factor_policy (role_name);

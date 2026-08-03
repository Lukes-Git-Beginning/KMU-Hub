-- Reverse of 000272. The tenant_id backfill on refresh_tokens is a pure join
-- from user_id, so it is trivially re-derivable — nothing is lost by dropping it.

DROP POLICY IF EXISTS tenant_isolation ON plugin_permissions;
ALTER TABLE plugin_permissions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE plugin_permissions DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON refresh_tokens;
ALTER TABLE refresh_tokens NO FORCE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_refresh_tokens_tenant_id;

ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS tenant_id;

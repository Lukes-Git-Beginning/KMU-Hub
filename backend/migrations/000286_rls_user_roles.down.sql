-- Reverse of 000286. tenant_id is a pure join from user_id, same as
-- refresh_tokens' — nothing is lost by dropping it.

DROP POLICY IF EXISTS tenant_isolation ON user_roles;
ALTER TABLE user_roles NO FORCE ROW LEVEL SECURITY;
ALTER TABLE user_roles DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_user_roles_tenant_id;

ALTER TABLE user_roles DROP COLUMN IF EXISTS tenant_id;

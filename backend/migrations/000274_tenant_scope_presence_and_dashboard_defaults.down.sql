-- Reverse of 000274. Collapsing per-tenant rows back into one global row is
-- lossy by definition: where tenants have diverged, only one setting survives.
-- The most recently edited row wins, so at least the newest administrator
-- decision is kept rather than an arbitrary one.

-- ---------------------------------------------------------------------------
-- dashboard_defaults
-- ---------------------------------------------------------------------------

DROP POLICY IF EXISTS tenant_isolation ON dashboard_defaults;
ALTER TABLE dashboard_defaults NO FORCE ROW LEVEL SECURITY;
ALTER TABLE dashboard_defaults DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_dashboard_defaults_tenant_role;

DELETE FROM dashboard_defaults
 WHERE id NOT IN (
     SELECT DISTINCT ON (role) id
     FROM dashboard_defaults
     ORDER BY role, updated_at DESC, id
 );

ALTER TABLE dashboard_defaults DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE dashboard_defaults ADD CONSTRAINT dashboard_defaults_role_key UNIQUE (role);
CREATE INDEX idx_dashboard_defaults_role ON dashboard_defaults(role);

-- ---------------------------------------------------------------------------
-- presence_config
-- ---------------------------------------------------------------------------

DROP POLICY IF EXISTS tenant_isolation ON presence_config;
ALTER TABLE presence_config NO FORCE ROW LEVEL SECURITY;
ALTER TABLE presence_config DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_presence_config_tenant;

DELETE FROM presence_config
 WHERE id NOT IN (
     SELECT id FROM presence_config ORDER BY updated_at DESC, id LIMIT 1
 );

ALTER TABLE presence_config DROP COLUMN IF EXISTS tenant_id;

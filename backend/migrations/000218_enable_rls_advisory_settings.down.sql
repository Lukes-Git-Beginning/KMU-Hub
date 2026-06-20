-- Reverse 000218: deactivate RLS on advisory_protocols + settings tables.
-- Columns and FKs are left intact (they predate this migration).

BEGIN;

DROP POLICY IF EXISTS tenant_isolation ON advisory_protocols;
ALTER TABLE advisory_protocols NO FORCE ROW LEVEL SECURITY;
ALTER TABLE advisory_protocols DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON tenant_settings;
ALTER TABLE tenant_settings NO FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON tenant_module_leads;
ALTER TABLE tenant_module_leads NO FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_module_leads DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON user_settings;
ALTER TABLE user_settings NO FORCE ROW LEVEL SECURITY;
ALTER TABLE user_settings DISABLE ROW LEVEL SECURITY;

COMMIT;

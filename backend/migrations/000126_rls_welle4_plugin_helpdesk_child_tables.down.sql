-- Reverse 000126: deactivate RLS, drop tenant FK + index, keep column NULLABLE.

BEGIN;

DROP POLICY IF EXISTS tenant_isolation ON plugin_execution_log;
ALTER TABLE plugin_execution_log NO FORCE ROW LEVEL SECURITY;
ALTER TABLE plugin_execution_log DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_plugin_execution_log_tenant_id;
ALTER TABLE plugin_execution_log DROP CONSTRAINT IF EXISTS fk_plugin_execution_log_tenant;
ALTER TABLE plugin_execution_log ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON plugin_kv_store;
ALTER TABLE plugin_kv_store NO FORCE ROW LEVEL SECURITY;
ALTER TABLE plugin_kv_store DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_plugin_kv_store_tenant_id;
ALTER TABLE plugin_kv_store DROP CONSTRAINT IF EXISTS fk_plugin_kv_store_tenant;
ALTER TABLE plugin_kv_store ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON ticket_messages;
ALTER TABLE ticket_messages NO FORCE ROW LEVEL SECURITY;
ALTER TABLE ticket_messages DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_ticket_messages_tenant_id;
ALTER TABLE ticket_messages DROP CONSTRAINT IF EXISTS fk_ticket_messages_tenant;
ALTER TABLE ticket_messages ALTER COLUMN tenant_id DROP NOT NULL;

COMMIT;

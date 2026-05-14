-- Sprint 4 Welle 4.2 — RLS Welle 4b: tenant-id-less child tables.
-- 2026-05-14
--
-- Three tables were deferred from welle 3 because they carry no tenant_id
-- of their own (mig 122 header Z. 20-28). All three have a parent FK that
-- carries tenant_id; we add the column, JOIN-backfill, then activate RLS
-- with the standard procedure rather than enable_tenant_rls_via_join().
--
-- Rationale: own column + index is faster at query time than the EXISTS
-- subquery emitted by enable_tenant_rls_via_join (mig 118 marks the latter
-- as a fallback for cases where ADD COLUMN is impractical). Plugin
-- execution log volume in particular benefits from a direct tenant_id index.

SET LOCAL row_security = off;

BEGIN;

-- ============================================================================
-- ticket_messages (mig 077) — FK ticket_id → tickets.tenant_id
-- ============================================================================

ALTER TABLE ticket_messages ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE ticket_messages tm
SET tenant_id = t.tenant_id
FROM tickets t
WHERE tm.ticket_id = t.id
  AND tm.tenant_id IS NULL;
-- orphaned rows (ticket deleted without cascade, should not exist) get sentinel
UPDATE ticket_messages SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE ticket_messages ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE ticket_messages
    ADD CONSTRAINT fk_ticket_messages_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_ticket_messages_tenant_id ON ticket_messages(tenant_id);
CALL enable_tenant_rls('ticket_messages');

-- ============================================================================
-- plugin_kv_store (mig 057) — FK installation_id → plugin_installations.tenant_id
-- ============================================================================

ALTER TABLE plugin_kv_store ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE plugin_kv_store pkv
SET tenant_id = pi.tenant_id
FROM plugin_installations pi
WHERE pkv.installation_id = pi.id
  AND pkv.tenant_id IS NULL;
UPDATE plugin_kv_store SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE plugin_kv_store ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE plugin_kv_store
    ADD CONSTRAINT fk_plugin_kv_store_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_plugin_kv_store_tenant_id ON plugin_kv_store(tenant_id);
CALL enable_tenant_rls('plugin_kv_store');

-- ============================================================================
-- plugin_execution_log (mig 057) — FK installation_id → plugin_installations.tenant_id
-- ============================================================================

ALTER TABLE plugin_execution_log ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE plugin_execution_log pel
SET tenant_id = pi.tenant_id
FROM plugin_installations pi
WHERE pel.installation_id = pi.id
  AND pel.tenant_id IS NULL;
UPDATE plugin_execution_log SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE plugin_execution_log ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE plugin_execution_log
    ADD CONSTRAINT fk_plugin_execution_log_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_plugin_execution_log_tenant_id ON plugin_execution_log(tenant_id);
CALL enable_tenant_rls('plugin_execution_log');

COMMIT;

-- Reverse of 000269. The backfilled tenant_id values stay — dropping NOT NULL
-- restores the old column definition, but re-nulling correct data would be a
-- loss, not a rollback.

DROP POLICY IF EXISTS tenant_isolation ON workflow_rules;
ALTER TABLE workflow_rules NO FORCE ROW LEVEL SECURITY;
ALTER TABLE workflow_rules DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON validation_rules;
ALTER TABLE validation_rules NO FORCE ROW LEVEL SECURITY;
ALTER TABLE validation_rules DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON email_contact_links;
ALTER TABLE email_contact_links NO FORCE ROW LEVEL SECURITY;
ALTER TABLE email_contact_links DISABLE ROW LEVEL SECURITY;
ALTER TABLE email_contact_links ALTER COLUMN tenant_id DROP NOT NULL;

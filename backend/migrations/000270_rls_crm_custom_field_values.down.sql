-- Reverse of 000270. Nothing to un-backfill: the join policy adds no column,
-- which is precisely why it was chosen over a tenant_id of its own.

DROP POLICY IF EXISTS tenant_isolation ON activity_custom_field_values;
ALTER TABLE activity_custom_field_values NO FORCE ROW LEVEL SECURITY;
ALTER TABLE activity_custom_field_values DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON deal_custom_field_values;
ALTER TABLE deal_custom_field_values NO FORCE ROW LEVEL SECURITY;
ALTER TABLE deal_custom_field_values DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON company_custom_field_values;
ALTER TABLE company_custom_field_values NO FORCE ROW LEVEL SECURITY;
ALTER TABLE company_custom_field_values DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON contact_custom_field_values;
ALTER TABLE contact_custom_field_values NO FORCE ROW LEVEL SECURITY;
ALTER TABLE contact_custom_field_values DISABLE ROW LEVEL SECURITY;

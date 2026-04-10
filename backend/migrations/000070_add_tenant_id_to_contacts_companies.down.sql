DROP INDEX IF EXISTS idx_companies_tenant;
ALTER TABLE companies DROP COLUMN IF EXISTS tenant_id;

DROP INDEX IF EXISTS idx_contacts_tenant;
ALTER TABLE contacts DROP COLUMN IF EXISTS tenant_id;

-- Rollback: drop tenant_id columns added in 000111
-- Reversed order (last added first). DROP COLUMN removes associated indexes automatically.

ALTER TABLE consent_records DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE activity_tags DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE deal_tags DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE company_tags DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE contact_tags DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE tags DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE ip_access_rules DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE password_history DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE password_policies DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE gdpr_erasure_log DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE gdpr_export_requests DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE vault_secrets DROP COLUMN IF EXISTS tenant_id;

-- Option-B Phase 2: add tenant_id to security, CRM-aux, and consent tables
-- Sentinel: 00000000-0000-0000-0000-000000000001 (existing single-tenant data)
-- All ALTERs use ADD COLUMN IF NOT EXISTS to be idempotent.
-- NOTE: audit_log already has tenant_id from 000106; NOT listed here.

-- vault_secrets
ALTER TABLE vault_secrets ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_vault_secrets_tenant_id ON vault_secrets(tenant_id);

-- gdpr_export_requests
ALTER TABLE gdpr_export_requests ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_gdpr_export_requests_tenant_id ON gdpr_export_requests(tenant_id);

-- gdpr_erasure_log
ALTER TABLE gdpr_erasure_log ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_gdpr_erasure_log_tenant_id ON gdpr_erasure_log(tenant_id);

-- password_policies
ALTER TABLE password_policies ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_password_policies_tenant_id ON password_policies(tenant_id);

-- password_history
ALTER TABLE password_history ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_password_history_tenant_id ON password_history(tenant_id);

-- ip_access_rules
ALTER TABLE ip_access_rules ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_ip_access_rules_tenant_id ON ip_access_rules(tenant_id);

-- tags
ALTER TABLE tags ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_tags_tenant_id ON tags(tenant_id);

-- contact_tags
ALTER TABLE contact_tags ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_contact_tags_tenant_id ON contact_tags(tenant_id);

-- company_tags
ALTER TABLE company_tags ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_company_tags_tenant_id ON company_tags(tenant_id);

-- deal_tags
ALTER TABLE deal_tags ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_deal_tags_tenant_id ON deal_tags(tenant_id);

-- activity_tags
ALTER TABLE activity_tags ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_activity_tags_tenant_id ON activity_tags(tenant_id);

-- consent_records
ALTER TABLE consent_records ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_consent_records_tenant_id ON consent_records(tenant_id);

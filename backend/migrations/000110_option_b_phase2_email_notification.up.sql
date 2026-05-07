-- Option-B Phase 2: add tenant_id to email, inbox, and notification tables
-- Sentinel: 00000000-0000-0000-0000-000000000001 (existing single-tenant data)
-- All ALTERs use ADD COLUMN IF NOT EXISTS to be idempotent.
-- NOTE: email_messages and inbox_messages already have tenant_id from 000106;
--       they are NOT listed here. Only their subtables / related tables are targeted.

-- email_accounts
ALTER TABLE email_accounts ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_email_accounts_tenant_id ON email_accounts(tenant_id);

-- email_folders
ALTER TABLE email_folders ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_email_folders_tenant_id ON email_folders(tenant_id);

-- email_attachments
ALTER TABLE email_attachments ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_email_attachments_tenant_id ON email_attachments(tenant_id);

-- email_signatures
ALTER TABLE email_signatures ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_email_signatures_tenant_id ON email_signatures(tenant_id);

-- email_contact_links
ALTER TABLE email_contact_links ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_email_contact_links_tenant_id ON email_contact_links(tenant_id);

-- team_inboxes
ALTER TABLE team_inboxes ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_team_inboxes_tenant_id ON team_inboxes(tenant_id);

-- team_inbox_members
ALTER TABLE team_inbox_members ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_team_inbox_members_tenant_id ON team_inbox_members(tenant_id);

-- routing_rules
ALTER TABLE routing_rules ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_routing_rules_tenant_id ON routing_rules(tenant_id);

-- notification_preferences
ALTER TABLE notification_preferences ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_notification_preferences_tenant_id ON notification_preferences(tenant_id);

-- notification_mutes
ALTER TABLE notification_mutes ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_notification_mutes_tenant_id ON notification_mutes(tenant_id);

-- notification_quiet_hours
ALTER TABLE notification_quiet_hours ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_notification_quiet_hours_tenant_id ON notification_quiet_hours(tenant_id);

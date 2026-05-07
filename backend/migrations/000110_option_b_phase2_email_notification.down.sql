-- Rollback Option-B Phase 2 email/notification tables: remove tenant_id
-- Reversed order relative to up.sql.
-- Indexes are dropped automatically with DROP COLUMN; explicit DROP INDEX
-- listed here for safety in case partial rollback leaves index orphans.

-- notification_quiet_hours
DROP INDEX IF EXISTS idx_notification_quiet_hours_tenant_id;
ALTER TABLE notification_quiet_hours DROP COLUMN IF EXISTS tenant_id;

-- notification_mutes
DROP INDEX IF EXISTS idx_notification_mutes_tenant_id;
ALTER TABLE notification_mutes DROP COLUMN IF EXISTS tenant_id;

-- notification_preferences
DROP INDEX IF EXISTS idx_notification_preferences_tenant_id;
ALTER TABLE notification_preferences DROP COLUMN IF EXISTS tenant_id;

-- routing_rules
DROP INDEX IF EXISTS idx_routing_rules_tenant_id;
ALTER TABLE routing_rules DROP COLUMN IF EXISTS tenant_id;

-- team_inbox_members
DROP INDEX IF EXISTS idx_team_inbox_members_tenant_id;
ALTER TABLE team_inbox_members DROP COLUMN IF EXISTS tenant_id;

-- team_inboxes
DROP INDEX IF EXISTS idx_team_inboxes_tenant_id;
ALTER TABLE team_inboxes DROP COLUMN IF EXISTS tenant_id;

-- email_contact_links
DROP INDEX IF EXISTS idx_email_contact_links_tenant_id;
ALTER TABLE email_contact_links DROP COLUMN IF EXISTS tenant_id;

-- email_signatures
DROP INDEX IF EXISTS idx_email_signatures_tenant_id;
ALTER TABLE email_signatures DROP COLUMN IF EXISTS tenant_id;

-- email_attachments
DROP INDEX IF EXISTS idx_email_attachments_tenant_id;
ALTER TABLE email_attachments DROP COLUMN IF EXISTS tenant_id;

-- email_folders
DROP INDEX IF EXISTS idx_email_folders_tenant_id;
ALTER TABLE email_folders DROP COLUMN IF EXISTS tenant_id;

-- email_accounts
DROP INDEX IF EXISTS idx_email_accounts_tenant_id;
ALTER TABLE email_accounts DROP COLUMN IF EXISTS tenant_id;

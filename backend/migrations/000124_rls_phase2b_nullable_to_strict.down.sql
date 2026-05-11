-- Revert: disable RLS and restore nullable tenant_id for the 41 tables activated in 000124.
-- No reverse backfill (destructive; data not recoverable). Rows keep their tenant_id values.

SET LOCAL row_security = off;

BEGIN;

-- Security
DROP POLICY IF EXISTS tenant_isolation ON ip_access_rules;
ALTER TABLE ip_access_rules DISABLE ROW LEVEL SECURITY;
ALTER TABLE ip_access_rules ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON password_history;
ALTER TABLE password_history DISABLE ROW LEVEL SECURITY;
ALTER TABLE password_history ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON password_policies;
ALTER TABLE password_policies DISABLE ROW LEVEL SECURITY;
ALTER TABLE password_policies ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON gdpr_erasure_log;
ALTER TABLE gdpr_erasure_log DISABLE ROW LEVEL SECURITY;
ALTER TABLE gdpr_erasure_log ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON gdpr_export_requests;
ALTER TABLE gdpr_export_requests DISABLE ROW LEVEL SECURITY;
ALTER TABLE gdpr_export_requests ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON vault_secrets;
ALTER TABLE vault_secrets DISABLE ROW LEVEL SECURITY;
ALTER TABLE vault_secrets ALTER COLUMN tenant_id DROP NOT NULL;

-- Notifications
DROP POLICY IF EXISTS tenant_isolation ON notification_quiet_hours;
ALTER TABLE notification_quiet_hours DISABLE ROW LEVEL SECURITY;
ALTER TABLE notification_quiet_hours ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON notification_mutes;
ALTER TABLE notification_mutes DISABLE ROW LEVEL SECURITY;
ALTER TABLE notification_mutes ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON notification_preferences;
ALTER TABLE notification_preferences DISABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences ALTER COLUMN tenant_id DROP NOT NULL;

-- Inbox
DROP POLICY IF EXISTS tenant_isolation ON routing_rules;
ALTER TABLE routing_rules DISABLE ROW LEVEL SECURITY;
ALTER TABLE routing_rules ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON team_inbox_members;
ALTER TABLE team_inbox_members DISABLE ROW LEVEL SECURITY;
ALTER TABLE team_inbox_members ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON team_inboxes;
ALTER TABLE team_inboxes DISABLE ROW LEVEL SECURITY;
ALTER TABLE team_inboxes ALTER COLUMN tenant_id DROP NOT NULL;

-- Email
DROP POLICY IF EXISTS tenant_isolation ON email_signatures;
ALTER TABLE email_signatures DISABLE ROW LEVEL SECURITY;
ALTER TABLE email_signatures ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON email_attachments;
ALTER TABLE email_attachments DISABLE ROW LEVEL SECURITY;
ALTER TABLE email_attachments ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON email_folders;
ALTER TABLE email_folders DISABLE ROW LEVEL SECURITY;
ALTER TABLE email_folders ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON email_accounts;
ALTER TABLE email_accounts DISABLE ROW LEVEL SECURITY;
ALTER TABLE email_accounts ALTER COLUMN tenant_id DROP NOT NULL;

-- Project sub-tables
DROP POLICY IF EXISTS tenant_isolation ON project_statuses;
ALTER TABLE project_statuses DISABLE ROW LEVEL SECURITY;
ALTER TABLE project_statuses ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON project_members;
ALTER TABLE project_members DISABLE ROW LEVEL SECURITY;
ALTER TABLE project_members ALTER COLUMN tenant_id DROP NOT NULL;

-- Task collaboration
DROP POLICY IF EXISTS tenant_isolation ON task_dependencies;
ALTER TABLE task_dependencies DISABLE ROW LEVEL SECURITY;
ALTER TABLE task_dependencies ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON task_files;
ALTER TABLE task_files DISABLE ROW LEVEL SECURITY;
ALTER TABLE task_files ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON task_activities;
ALTER TABLE task_activities DISABLE ROW LEVEL SECURITY;
ALTER TABLE task_activities ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON task_comments;
ALTER TABLE task_comments DISABLE ROW LEVEL SECURITY;
ALTER TABLE task_comments ALTER COLUMN tenant_id DROP NOT NULL;

-- Resources
DROP POLICY IF EXISTS tenant_isolation ON resource_bookings;
ALTER TABLE resource_bookings DISABLE ROW LEVEL SECURITY;
ALTER TABLE resource_bookings ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON resource_tags;
ALTER TABLE resource_tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE resource_tags ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON resources;
ALTER TABLE resources DISABLE ROW LEVEL SECURITY;
ALTER TABLE resources ALTER COLUMN tenant_id DROP NOT NULL;

-- Meetings
DROP POLICY IF EXISTS tenant_isolation ON meeting_action_items;
ALTER TABLE meeting_action_items DISABLE ROW LEVEL SECURITY;
ALTER TABLE meeting_action_items ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON meeting_notes;
ALTER TABLE meeting_notes DISABLE ROW LEVEL SECURITY;
ALTER TABLE meeting_notes ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON meeting_attendees;
ALTER TABLE meeting_attendees DISABLE ROW LEVEL SECURITY;
ALTER TABLE meeting_attendees ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON meetings;
ALTER TABLE meetings DISABLE ROW LEVEL SECURITY;
ALTER TABLE meetings ALTER COLUMN tenant_id DROP NOT NULL;

-- Events
DROP POLICY IF EXISTS tenant_isolation ON event_reminders;
ALTER TABLE event_reminders DISABLE ROW LEVEL SECURITY;
ALTER TABLE event_reminders ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON event_exceptions;
ALTER TABLE event_exceptions DISABLE ROW LEVEL SECURITY;
ALTER TABLE event_exceptions ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON event_attendees;
ALTER TABLE event_attendees DISABLE ROW LEVEL SECURITY;
ALTER TABLE event_attendees ALTER COLUMN tenant_id DROP NOT NULL;

-- Calendar
DROP POLICY IF EXISTS tenant_isolation ON user_calendar_preferences;
ALTER TABLE user_calendar_preferences DISABLE ROW LEVEL SECURITY;
ALTER TABLE user_calendar_preferences ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON event_categories;
ALTER TABLE event_categories DISABLE ROW LEVEL SECURITY;
ALTER TABLE event_categories ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON calendar_members;
ALTER TABLE calendar_members DISABLE ROW LEVEL SECURITY;
ALTER TABLE calendar_members ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON calendars;
ALTER TABLE calendars DISABLE ROW LEVEL SECURITY;
ALTER TABLE calendars ALTER COLUMN tenant_id DROP NOT NULL;

-- CRM junction tables
DROP POLICY IF EXISTS tenant_isolation ON activity_tags;
ALTER TABLE activity_tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE activity_tags ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON deal_tags;
ALTER TABLE deal_tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE deal_tags ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON company_tags;
ALTER TABLE company_tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE company_tags ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON contact_tags;
ALTER TABLE contact_tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE contact_tags ALTER COLUMN tenant_id DROP NOT NULL;

DROP POLICY IF EXISTS tenant_isolation ON tags;
ALTER TABLE tags DISABLE ROW LEVEL SECURITY;
ALTER TABLE tags ALTER COLUMN tenant_id DROP NOT NULL;

COMMIT;

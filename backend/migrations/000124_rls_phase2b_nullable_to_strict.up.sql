-- Sprint 4 Welle 3.1B — RLS Phase 2b: backfill + NOT NULL + activate for 41 NULLABLE long-tail tables
-- (3 tables deferred to Welle 4: integration_configs, bexio_sync_configs, lexware_sync_configs —
--  their ON CONFLICT (platform) unique constraint conflicts with per-tenant rows; needs constraint
--  change to (platform, tenant_id) first.)
-- 2026-05-11
--
-- Builds on:
--   - 000109 (ADD COLUMN tenant_id on calendar/work/resource/task-collaboration tables)
--   - 000110 (ADD COLUMN tenant_id on email/inbox/notification tables)
--   - 000111 (ADD COLUMN tenant_id on security/CRM-aux tables)
--   - 000118 (enable_tenant_rls procedure + helper functions)
--   - 000122 (RLS on NOT-NULL long-tail tables — this migration handles the NULLABLE remainder)
--
-- Backfill strategies:
--   JOIN-backfill:  child table JOINs parent via FK to copy parent.tenant_id
--   user-backfill:  child table JOINs users to copy users.tenant_id
--   sentinel:       '00000000-0000-0000-0000-000000000001' for standalone tables without
--                   a reliable FK (password_policies, vault_secrets, ip_access_rules)
--
-- The migration tx runs with row_security off so the ALTER TABLE and CALL statements
-- are not themselves subject to policies not yet created.

SET LOCAL row_security = off;

BEGIN;

-- ============================================================================
-- CRM Junction Tables (mig 000006 + 000111)
-- ============================================================================

-- tags — standalone, sentinel fallback
UPDATE tags SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE tags ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('tags');

-- contact_tags — JOIN via contacts
UPDATE contact_tags ct
SET tenant_id = c.tenant_id
FROM contacts c
WHERE ct.contact_id = c.id AND ct.tenant_id IS NULL;
-- orphaned rows (contact deleted without cascade) get sentinel
UPDATE contact_tags SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE contact_tags ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('contact_tags');

-- company_tags — JOIN via companies
UPDATE company_tags ct
SET tenant_id = c.tenant_id
FROM companies c
WHERE ct.company_id = c.id AND ct.tenant_id IS NULL;
UPDATE company_tags SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE company_tags ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('company_tags');

-- deal_tags — JOIN via deals
UPDATE deal_tags dt
SET tenant_id = d.tenant_id
FROM deals d
WHERE dt.deal_id = d.id AND dt.tenant_id IS NULL;
UPDATE deal_tags SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE deal_tags ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('deal_tags');

-- activity_tags — JOIN via activities
UPDATE activity_tags at
SET tenant_id = a.tenant_id
FROM activities a
WHERE at.activity_id = a.id AND at.tenant_id IS NULL;
UPDATE activity_tags SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE activity_tags ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('activity_tags');

-- ============================================================================
-- Calendar Module (mig 000032 + 000109)
-- ============================================================================

-- calendars — standalone (no parent FK, owner_id references users)
UPDATE calendars cal
SET tenant_id = u.tenant_id
FROM users u
WHERE cal.owner_id = u.id AND cal.tenant_id IS NULL;
UPDATE calendars SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE calendars ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('calendars');

-- calendar_members — JOIN via calendars
UPDATE calendar_members cm
SET tenant_id = c.tenant_id
FROM calendars c
WHERE cm.calendar_id = c.id AND cm.tenant_id IS NULL;
UPDATE calendar_members SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE calendar_members ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('calendar_members');

-- event_categories — user-scoped
UPDATE event_categories ec
SET tenant_id = u.tenant_id
FROM users u
WHERE ec.user_id = u.id AND ec.tenant_id IS NULL;
UPDATE event_categories SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE event_categories ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('event_categories');

-- user_calendar_preferences — user-scoped
UPDATE user_calendar_preferences ucp
SET tenant_id = u.tenant_id
FROM users u
WHERE ucp.user_id = u.id AND ucp.tenant_id IS NULL;
UPDATE user_calendar_preferences SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE user_calendar_preferences ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('user_calendar_preferences');

-- ============================================================================
-- Events Module (mig 000033 + 000109)
-- ============================================================================

-- event_attendees — JOIN via calendar_events
UPDATE event_attendees ea
SET tenant_id = e.tenant_id
FROM calendar_events e
WHERE ea.event_id = e.id AND ea.tenant_id IS NULL;
UPDATE event_attendees SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE event_attendees ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('event_attendees');

-- event_exceptions — JOIN via calendar_events
UPDATE event_exceptions ex
SET tenant_id = e.tenant_id
FROM calendar_events e
WHERE ex.event_id = e.id AND ex.tenant_id IS NULL;
UPDATE event_exceptions SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE event_exceptions ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('event_exceptions');

-- event_reminders — JOIN via calendar_events
UPDATE event_reminders er
SET tenant_id = e.tenant_id
FROM calendar_events e
WHERE er.event_id = e.id AND er.tenant_id IS NULL;
UPDATE event_reminders SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE event_reminders ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('event_reminders');

-- ============================================================================
-- Meetings Module (mig 000037 + 000109)
-- ============================================================================

-- meetings — standalone (organizer_id references users)
UPDATE meetings m
SET tenant_id = u.tenant_id
FROM users u
WHERE m.organizer_id = u.id AND m.tenant_id IS NULL;
UPDATE meetings SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE meetings ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('meetings');

-- meeting_attendees — JOIN via meetings
UPDATE meeting_attendees ma
SET tenant_id = m.tenant_id
FROM meetings m
WHERE ma.meeting_id = m.id AND ma.tenant_id IS NULL;
UPDATE meeting_attendees SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE meeting_attendees ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('meeting_attendees');

-- meeting_notes — JOIN via meetings
UPDATE meeting_notes mn
SET tenant_id = m.tenant_id
FROM meetings m
WHERE mn.meeting_id = m.id AND mn.tenant_id IS NULL;
UPDATE meeting_notes SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE meeting_notes ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('meeting_notes');

-- meeting_action_items — already has tenant_id NOT NULL from earlier welle (verified in mig 000109);
-- kept here as a safety guard in case any NULL slipped through a race.
UPDATE meeting_action_items SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE meeting_action_items ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('meeting_action_items');

-- ============================================================================
-- Resources Module (mig 000034 + 000109)
-- ============================================================================

-- resources — standalone (tenant_id set by application layer via booking organizer)
UPDATE resources r
SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE resources ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('resources');

-- resource_tags — JOIN via resources
UPDATE resource_tags rt
SET tenant_id = r.tenant_id
FROM resources r
WHERE rt.resource_id = r.id AND rt.tenant_id IS NULL;
UPDATE resource_tags SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE resource_tags ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('resource_tags');

-- resource_bookings — JOIN via resources (resource_bookings.tenant_id already set in many rows)
UPDATE resource_bookings rb
SET tenant_id = r.tenant_id
FROM resources r
WHERE rb.resource_id = r.id AND rb.tenant_id IS NULL;
UPDATE resource_bookings SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE resource_bookings ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('resource_bookings');

-- ============================================================================
-- Task Collaboration (mig 000026 + 000109)
-- ============================================================================

-- task_comments — JOIN via tasks → projects
UPDATE task_comments tc
SET tenant_id = p.tenant_id
FROM tasks t JOIN projects p ON t.project_id = p.id
WHERE tc.task_id = t.id AND tc.tenant_id IS NULL;
UPDATE task_comments SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE task_comments ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('task_comments');

-- task_activities — JOIN via tasks → projects
UPDATE task_activities ta
SET tenant_id = p.tenant_id
FROM tasks t JOIN projects p ON t.project_id = p.id
WHERE ta.task_id = t.id AND ta.tenant_id IS NULL;
UPDATE task_activities SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE task_activities ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('task_activities');

-- task_files — JOIN via tasks → projects
UPDATE task_files tf
SET tenant_id = p.tenant_id
FROM tasks t JOIN projects p ON t.project_id = p.id
WHERE tf.task_id = t.id AND tf.tenant_id IS NULL;
UPDATE task_files SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE task_files ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('task_files');

-- task_dependencies — JOIN via source_task_id → tasks → projects
UPDATE task_dependencies td
SET tenant_id = p.tenant_id
FROM tasks t JOIN projects p ON t.project_id = p.id
WHERE td.source_task_id = t.id AND td.tenant_id IS NULL;
UPDATE task_dependencies SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE task_dependencies ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('task_dependencies');

-- ============================================================================
-- Project Sub-Tables (mig 000024 + 000109)
-- ============================================================================

-- project_members — JOIN via projects
UPDATE project_members pm
SET tenant_id = p.tenant_id
FROM projects p
WHERE pm.project_id = p.id AND pm.tenant_id IS NULL;
UPDATE project_members SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE project_members ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('project_members');

-- project_statuses — JOIN via projects
UPDATE project_statuses ps
SET tenant_id = p.tenant_id
FROM projects p
WHERE ps.project_id = p.id AND ps.tenant_id IS NULL;
UPDATE project_statuses SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE project_statuses ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('project_statuses');

-- ============================================================================
-- Email Module (mig 000041 + 000110)
-- ============================================================================

-- email_accounts — standalone (user_id references users)
UPDATE email_accounts ea
SET tenant_id = u.tenant_id
FROM users u
WHERE ea.user_id = u.id AND ea.tenant_id IS NULL;
UPDATE email_accounts SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE email_accounts ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('email_accounts');

-- email_folders — JOIN via email_accounts
UPDATE email_folders ef
SET tenant_id = ea.tenant_id
FROM email_accounts ea
WHERE ef.account_id = ea.id AND ef.tenant_id IS NULL;
UPDATE email_folders SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE email_folders ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('email_folders');

-- email_attachments — JOIN via email_messages (tenant_id already NOT NULL on email_messages)
UPDATE email_attachments att
SET tenant_id = msg.tenant_id
FROM email_messages msg
WHERE att.message_id = msg.id AND att.tenant_id IS NULL;
UPDATE email_attachments SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE email_attachments ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('email_attachments');

-- email_signatures — user-scoped
UPDATE email_signatures es
SET tenant_id = u.tenant_id
FROM users u
WHERE es.user_id = u.id AND es.tenant_id IS NULL;
UPDATE email_signatures SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE email_signatures ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('email_signatures');

-- ============================================================================
-- Inbox Module (mig 000047 + 000110)
-- ============================================================================

-- team_inboxes — standalone
UPDATE team_inboxes SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE team_inboxes ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('team_inboxes');

-- team_inbox_members — JOIN via team_inboxes
UPDATE team_inbox_members tim
SET tenant_id = ti.tenant_id
FROM team_inboxes ti
WHERE tim.team_inbox_id = ti.id AND tim.tenant_id IS NULL;
UPDATE team_inbox_members SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE team_inbox_members ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('team_inbox_members');

-- routing_rules — standalone
UPDATE routing_rules SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE routing_rules ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('routing_rules');

-- ============================================================================
-- Notification Module (mig 000022 + 000110)
-- ============================================================================

-- notification_preferences — user-scoped
UPDATE notification_preferences np
SET tenant_id = u.tenant_id
FROM users u
WHERE np.user_id = u.id AND np.tenant_id IS NULL;
UPDATE notification_preferences SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE notification_preferences ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('notification_preferences');

-- notification_mutes — user-scoped
UPDATE notification_mutes nm
SET tenant_id = u.tenant_id
FROM users u
WHERE nm.user_id = u.id AND nm.tenant_id IS NULL;
UPDATE notification_mutes SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE notification_mutes ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('notification_mutes');

-- notification_quiet_hours — user-scoped
UPDATE notification_quiet_hours nqh
SET tenant_id = u.tenant_id
FROM users u
WHERE nqh.user_id = u.id AND nqh.tenant_id IS NULL;
UPDATE notification_quiet_hours SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE notification_quiet_hours ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('notification_quiet_hours');

-- ============================================================================
-- Security Module (mig 000039 + 000111)
-- ============================================================================

-- vault_secrets — sentinel (platform admin secrets, no reliable user/tenant FK at creation time)
UPDATE vault_secrets SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE vault_secrets ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('vault_secrets');

-- gdpr_export_requests — user-scoped
UPDATE gdpr_export_requests ger
SET tenant_id = u.tenant_id
FROM users u
WHERE ger.user_id = u.id AND ger.tenant_id IS NULL;
UPDATE gdpr_export_requests SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE gdpr_export_requests ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('gdpr_export_requests');

-- gdpr_erasure_log — JOIN via original_user_id (users, ON DELETE SET NULL pattern)
-- users may have been erased; sentinel for orphaned rows
UPDATE gdpr_erasure_log gel
SET tenant_id = u.tenant_id
FROM users u
WHERE gel.original_user_id = u.id AND gel.tenant_id IS NULL;
UPDATE gdpr_erasure_log SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE gdpr_erasure_log ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('gdpr_erasure_log');

-- password_policies — one row per tenant; sentinel for existing single-tenant rows
UPDATE password_policies SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE password_policies ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('password_policies');

-- password_history — user-scoped
UPDATE password_history ph
SET tenant_id = u.tenant_id
FROM users u
WHERE ph.user_id = u.id AND ph.tenant_id IS NULL;
UPDATE password_history SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE password_history ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('password_history');

-- ip_access_rules — standalone (admin-created, sentinel for pre-multi-tenant rows)
UPDATE ip_access_rules SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid WHERE tenant_id IS NULL;
ALTER TABLE ip_access_rules ALTER COLUMN tenant_id SET NOT NULL;
CALL enable_tenant_rls('ip_access_rules');

COMMIT;

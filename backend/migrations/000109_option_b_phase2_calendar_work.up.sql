-- Option-B Phase 2: add tenant_id to calendar, work, resource, and task-collaboration tables
-- Sentinel: 00000000-0000-0000-0000-000000000001 (existing single-tenant data)
-- All ALTERs use ADD COLUMN IF NOT EXISTS to be idempotent.

-- calendars
ALTER TABLE calendars ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_calendars_tenant_id ON calendars(tenant_id);

-- calendar_members
ALTER TABLE calendar_members ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_calendar_members_tenant_id ON calendar_members(tenant_id);

-- event_categories
ALTER TABLE event_categories ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_event_categories_tenant_id ON event_categories(tenant_id);

-- event_attendees
ALTER TABLE event_attendees ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_event_attendees_tenant_id ON event_attendees(tenant_id);

-- event_exceptions
ALTER TABLE event_exceptions ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_event_exceptions_tenant_id ON event_exceptions(tenant_id);

-- event_reminders
ALTER TABLE event_reminders ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_event_reminders_tenant_id ON event_reminders(tenant_id);

-- user_calendar_preferences
ALTER TABLE user_calendar_preferences ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_user_calendar_preferences_tenant_id ON user_calendar_preferences(tenant_id);

-- meetings
ALTER TABLE meetings ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_meetings_tenant_id ON meetings(tenant_id);

-- meeting_attendees
ALTER TABLE meeting_attendees ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_meeting_attendees_tenant_id ON meeting_attendees(tenant_id);

-- meeting_notes
ALTER TABLE meeting_notes ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_meeting_notes_tenant_id ON meeting_notes(tenant_id);

-- meeting_action_items
ALTER TABLE meeting_action_items ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_meeting_action_items_tenant_id ON meeting_action_items(tenant_id);

-- recording_consents
-- NOTE: recordings table already has tenant_id from 000106 (ADD COLUMN IF NOT EXISTS is safe).
ALTER TABLE recording_consents ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_recording_consents_tenant_id ON recording_consents(tenant_id);

-- resources
ALTER TABLE resources ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_resources_tenant_id ON resources(tenant_id);

-- resource_tags
ALTER TABLE resource_tags ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_resource_tags_tenant_id ON resource_tags(tenant_id);

-- resource_bookings
ALTER TABLE resource_bookings ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_resource_bookings_tenant_id ON resource_bookings(tenant_id);

-- task_comments
ALTER TABLE task_comments ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_task_comments_tenant_id ON task_comments(tenant_id);

-- task_activities
ALTER TABLE task_activities ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_task_activities_tenant_id ON task_activities(tenant_id);

-- task_files
ALTER TABLE task_files ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_task_files_tenant_id ON task_files(tenant_id);

-- task_dependencies
ALTER TABLE task_dependencies ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_task_dependencies_tenant_id ON task_dependencies(tenant_id);

-- project_members
ALTER TABLE project_members ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_project_members_tenant_id ON project_members(tenant_id);

-- project_statuses
ALTER TABLE project_statuses ADD COLUMN IF NOT EXISTS tenant_id UUID;
CREATE INDEX IF NOT EXISTS idx_project_statuses_tenant_id ON project_statuses(tenant_id);

-- Composite index for recordings: meeting_id + tenant_id hot-path join
-- (Risk mitigation: recordings FK to meetings, tenant-scoped lookup critical)
CREATE INDEX IF NOT EXISTS idx_recordings_meeting_id_tenant_id ON recordings(meeting_id, tenant_id);

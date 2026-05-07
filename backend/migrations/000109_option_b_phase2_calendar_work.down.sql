-- Rollback Option-B Phase 2 calendar/work tables: remove tenant_id
-- Reversed order relative to up.sql.
-- Indexes are dropped automatically with DROP COLUMN; explicit DROP INDEX
-- listed here for safety in case partial rollback leaves index orphans.

-- Composite index on recordings (added in up.sql)
DROP INDEX IF EXISTS idx_recordings_meeting_id_tenant_id;

-- project_statuses
DROP INDEX IF EXISTS idx_project_statuses_tenant_id;
ALTER TABLE project_statuses DROP COLUMN IF EXISTS tenant_id;

-- project_members
DROP INDEX IF EXISTS idx_project_members_tenant_id;
ALTER TABLE project_members DROP COLUMN IF EXISTS tenant_id;

-- task_dependencies
DROP INDEX IF EXISTS idx_task_dependencies_tenant_id;
ALTER TABLE task_dependencies DROP COLUMN IF EXISTS tenant_id;

-- task_files
DROP INDEX IF EXISTS idx_task_files_tenant_id;
ALTER TABLE task_files DROP COLUMN IF EXISTS tenant_id;

-- task_activities
DROP INDEX IF EXISTS idx_task_activities_tenant_id;
ALTER TABLE task_activities DROP COLUMN IF EXISTS tenant_id;

-- task_comments
DROP INDEX IF EXISTS idx_task_comments_tenant_id;
ALTER TABLE task_comments DROP COLUMN IF EXISTS tenant_id;

-- resource_bookings
DROP INDEX IF EXISTS idx_resource_bookings_tenant_id;
ALTER TABLE resource_bookings DROP COLUMN IF EXISTS tenant_id;

-- resource_tags
DROP INDEX IF EXISTS idx_resource_tags_tenant_id;
ALTER TABLE resource_tags DROP COLUMN IF EXISTS tenant_id;

-- resources
DROP INDEX IF EXISTS idx_resources_tenant_id;
ALTER TABLE resources DROP COLUMN IF EXISTS tenant_id;

-- recording_consents
DROP INDEX IF EXISTS idx_recording_consents_tenant_id;
ALTER TABLE recording_consents DROP COLUMN IF EXISTS tenant_id;

-- meeting_action_items
DROP INDEX IF EXISTS idx_meeting_action_items_tenant_id;
ALTER TABLE meeting_action_items DROP COLUMN IF EXISTS tenant_id;

-- meeting_notes
DROP INDEX IF EXISTS idx_meeting_notes_tenant_id;
ALTER TABLE meeting_notes DROP COLUMN IF EXISTS tenant_id;

-- meeting_attendees
DROP INDEX IF EXISTS idx_meeting_attendees_tenant_id;
ALTER TABLE meeting_attendees DROP COLUMN IF EXISTS tenant_id;

-- meetings
DROP INDEX IF EXISTS idx_meetings_tenant_id;
ALTER TABLE meetings DROP COLUMN IF EXISTS tenant_id;

-- user_calendar_preferences
DROP INDEX IF EXISTS idx_user_calendar_preferences_tenant_id;
ALTER TABLE user_calendar_preferences DROP COLUMN IF EXISTS tenant_id;

-- event_reminders
DROP INDEX IF EXISTS idx_event_reminders_tenant_id;
ALTER TABLE event_reminders DROP COLUMN IF EXISTS tenant_id;

-- event_exceptions
DROP INDEX IF EXISTS idx_event_exceptions_tenant_id;
ALTER TABLE event_exceptions DROP COLUMN IF EXISTS tenant_id;

-- event_attendees
DROP INDEX IF EXISTS idx_event_attendees_tenant_id;
ALTER TABLE event_attendees DROP COLUMN IF EXISTS tenant_id;

-- event_categories
DROP INDEX IF EXISTS idx_event_categories_tenant_id;
ALTER TABLE event_categories DROP COLUMN IF EXISTS tenant_id;

-- calendar_members
DROP INDEX IF EXISTS idx_calendar_members_tenant_id;
ALTER TABLE calendar_members DROP COLUMN IF EXISTS tenant_id;

-- calendars
DROP INDEX IF EXISTS idx_calendars_tenant_id;
ALTER TABLE calendars DROP COLUMN IF EXISTS tenant_id;

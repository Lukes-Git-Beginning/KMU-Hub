-- Add COMMENT ON metadata to the most-used tables and their key columns.
-- Purpose: self-documenting schema for AI agents (via MCP) and humans (via psql \d+, DBeaver).
-- Pattern: explain what the entity is, when rows change, enum semantics, and non-obvious relationships.

-- ============================================================================
-- users
-- ============================================================================
COMMENT ON TABLE users IS
'Application users — staff who log in to Cosmi. Not to be confused with contacts (CRM entities).
 Authentication via password_hash + optional 2FA (see two_factor_policy, recovery_codes).
 is_active=false disables login without deleting the row.';
COMMENT ON COLUMN users.email IS
'Unique login identifier. Normalised by application layer (lowercase).';
COMMENT ON COLUMN users.is_active IS
'Soft disable for login. Admins set to false on offboarding. Historical FKs remain intact.';

-- ============================================================================
-- companies
-- ============================================================================
COMMENT ON TABLE companies IS
'B2B account entities — customer organisations, prospects, partners, vendors.
 Contacts reference companies via contacts.company_id (nullable: free-floating contacts allowed).
 Multi-tenancy: tenant_id added in migration 000070.';
COMMENT ON COLUMN companies.domain IS
'Primary web domain (e.g. "example.com"). Used for auto-matching email senders to companies.';
COMMENT ON COLUMN companies.industry IS
'Industry label — free text for now. Industry_templates table (migration ~060) may constrain this later.';

-- ============================================================================
-- contacts
-- ============================================================================
COMMENT ON TABLE contacts IS
'Individual CRM contacts — leads, customers, partners. Attached to a company via company_id (optional).
 No soft-delete: hard DELETE with ON DELETE SET NULL on foreign tables. Preserves history in activities/deals.
 Multi-tenancy: tenant_id added in migration 000070.';
COMMENT ON COLUMN contacts.email IS
'Unique across all contacts (case-insensitive). NULL allowed for contacts without email.';
COMMENT ON COLUMN contacts.company_id IS
'Optional FK to companies. On company delete: set to NULL (contact remains as free-floating).';
COMMENT ON COLUMN contacts.position IS
'Job title / role at the company. Free text.';

-- ============================================================================
-- deals
-- ============================================================================
COMMENT ON TABLE deals IS
'Sales opportunities moving through pipeline_stages. Attached to contact_id and/or company_id (both optional).
 Revenue = SUM(value) WHERE stage is "closed won" (determined by pipeline_stages metadata, not a boolean here).
 closed_at timestamp = when the deal was marked closed (won OR lost). Stage history in deal_stage_history.';
COMMENT ON COLUMN deals.value IS
'Deal value in the currency specified by deals.currency. DECIMAL(15,2) — precise money.';
COMMENT ON COLUMN deals.currency IS
'ISO 4217 code. Default EUR per CLAUDE.md locale default (de-DE).';
COMMENT ON COLUMN deals.stage_id IS
'Current pipeline stage. Moving a deal updates this FK and appends to deal_stage_history.';
COMMENT ON COLUMN deals.owner_id IS
'Sales rep responsible for the deal. Distinct from created_by. NULL on user offboarding.';
COMMENT ON COLUMN deals.expected_close_date IS
'Forecast target date. Used for pipeline-value-by-quarter reports.';

-- ============================================================================
-- activities
-- ============================================================================
COMMENT ON TABLE activities IS
'CRM interactions and to-dos: calls, meetings, notes, emails, tasks (see activity_type enum).
 Polymorphic attachment: contact_id OR company_id OR deal_id (any combination).
 is_completed + completed_at for task-style activities; calls/notes are typically completed on creation.';
COMMENT ON COLUMN activities.activity_type IS
'Enum: call | meeting | note | email | task. Drives UI rendering and filter presets.';
COMMENT ON COLUMN activities.due_date IS
'For task-type activities: when it is due. NULL for logged past activities (notes, completed calls).';
COMMENT ON COLUMN activities.is_completed IS
'Mirrors completed_at IS NOT NULL but indexed directly for open-task filters.';

-- ============================================================================
-- projects
-- ============================================================================
COMMENT ON TABLE projects IS
'Project-management projects (distinct from CRM deals). Contain tasks and optional members.
 project_key is a short prefix (e.g. "ENG") used for human-readable task identifiers like "ENG-42".
 archived_at replaces hard-delete to preserve task history.';
COMMENT ON COLUMN projects.project_key IS
'Short uppercase prefix for task numbering. Unique among non-archived projects.';
COMMENT ON COLUMN projects.next_task_number IS
'Counter for generating the next task_number within this project. Increments atomically on task insert.';
COMMENT ON COLUMN projects.archived_at IS
'Soft-archive timestamp. NULL = active. Archived projects keep their project_key free for reuse.';

-- ============================================================================
-- tasks
-- ============================================================================
COMMENT ON TABLE tasks IS
'Work items within projects. Hierarchical via parent_task_id (depth cached for efficient tree queries).
 Human-readable id = projects.project_key || "-" || tasks.task_number (unique per project).
 German full-text search via search_vector (tsvector, trigger-maintained).';
COMMENT ON COLUMN tasks.task_number IS
'Sequential number within the project. Combined with project.project_key for display (e.g. "ENG-42").';
COMMENT ON COLUMN tasks.priority IS
'Enum-like: urgent | high | medium (default) | low. Enforced by CHECK constraint, not a PG enum type.';
COMMENT ON COLUMN tasks.parent_task_id IS
'Optional parent for subtasks. ON DELETE CASCADE: deleting a parent removes all descendants.';
COMMENT ON COLUMN tasks.search_vector IS
'German-language tsvector auto-maintained by tasks_search_vector_update trigger.
 Title weighted A, description weighted B. Queried via GIN idx_tasks_search.';

-- ============================================================================
-- meetings
-- ============================================================================
COMMENT ON TABLE meetings IS
'Video/in-person meetings with structured attendees, notes, and action items.
 scheduled_start/end = planned time; actual_start/end populated when meeting goes live via LiveKit.
 Linked to a calendar_events row via calendar_event_id (bidirectional, one-way FK only for compatibility).';
COMMENT ON COLUMN meetings.status IS
'scheduled | in_progress | completed | cancelled. Transitions: scheduled -> in_progress (on LiveKit join) -> completed.';
COMMENT ON COLUMN meetings.room_name IS
'LiveKit room identifier. Null until the meeting is started.';
COMMENT ON COLUMN meetings.recurring_meeting_id IS
'Self-FK to the "template" meeting that spawned this occurrence. NULL for one-off meetings.';

-- ============================================================================
-- calendar_events
-- ============================================================================
COMMENT ON TABLE calendar_events IS
'RFC 5545 compatible calendar events. Recurrence via rrule string; per-occurrence overrides in event_exceptions.
 timezone defaults to Europe/Berlin per de-DE locale.
 has_video_call + livekit_room_name integrate with the meetings subsystem.';
COMMENT ON COLUMN calendar_events.rrule IS
'RFC 5545 RRULE string (e.g. "FREQ=WEEKLY;BYDAY=MO,WE"). NULL = non-recurring event.';
COMMENT ON COLUMN calendar_events.recurrence_end IS
'UNTIL boundary for recurring events. NULL = indefinite recurrence.';
COMMENT ON COLUMN calendar_events.timezone IS
'IANA zone name. Default Europe/Berlin matches de-DE default locale.';

-- ============================================================================
-- dialer_campaigns
-- ============================================================================
COMMENT ON TABLE dialer_campaigns IS
'Outbound dialer campaigns (ZFA-pilot module). Each campaign groups contacts + assigned agents + call settings.
 Lifecycle: draft -> active (started_at set) -> paused/completed (completed_at set) -> archived.
 tenant_id is NOT NULL (dialer was the first fully multi-tenant-aware module).';
COMMENT ON COLUMN dialer_campaigns.mode IS
'Dialing strategy: preview (manual confirm) | power (1:1 auto-dial) | predictive (over-dial ratio).
 Preview is the default and only mode used in ZFA Pilot 0.';
COMMENT ON COLUMN dialer_campaigns.settings IS
'JSONB: dial ratio, retry intervals, time-of-day windows, voicemail rules. Schema evolves per mode.';
COMMENT ON COLUMN dialer_campaigns.assigned_agent_ids IS
'UUID[] of users(id) authorised to work this campaign. GIN index not needed at current scale.';
COMMENT ON COLUMN dialer_campaigns.contact_count IS
'Denormalised count of dialer_campaign_contacts rows. Updated via trigger or service layer.';

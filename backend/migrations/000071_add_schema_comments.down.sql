-- Remove comments added in 000071.

-- users
COMMENT ON TABLE users IS NULL;
COMMENT ON COLUMN users.email IS NULL;
COMMENT ON COLUMN users.is_active IS NULL;

-- companies
COMMENT ON TABLE companies IS NULL;
COMMENT ON COLUMN companies.domain IS NULL;
COMMENT ON COLUMN companies.industry IS NULL;

-- contacts
COMMENT ON TABLE contacts IS NULL;
COMMENT ON COLUMN contacts.email IS NULL;
COMMENT ON COLUMN contacts.company_id IS NULL;
COMMENT ON COLUMN contacts.position IS NULL;

-- deals
COMMENT ON TABLE deals IS NULL;
COMMENT ON COLUMN deals.value IS NULL;
COMMENT ON COLUMN deals.currency IS NULL;
COMMENT ON COLUMN deals.stage_id IS NULL;
COMMENT ON COLUMN deals.owner_id IS NULL;
COMMENT ON COLUMN deals.expected_close_date IS NULL;

-- activities
COMMENT ON TABLE activities IS NULL;
COMMENT ON COLUMN activities.activity_type IS NULL;
COMMENT ON COLUMN activities.due_date IS NULL;
COMMENT ON COLUMN activities.is_completed IS NULL;

-- projects
COMMENT ON TABLE projects IS NULL;
COMMENT ON COLUMN projects.project_key IS NULL;
COMMENT ON COLUMN projects.next_task_number IS NULL;
COMMENT ON COLUMN projects.archived_at IS NULL;

-- tasks
COMMENT ON TABLE tasks IS NULL;
COMMENT ON COLUMN tasks.task_number IS NULL;
COMMENT ON COLUMN tasks.priority IS NULL;
COMMENT ON COLUMN tasks.parent_task_id IS NULL;
COMMENT ON COLUMN tasks.search_vector IS NULL;

-- meetings
COMMENT ON TABLE meetings IS NULL;
COMMENT ON COLUMN meetings.status IS NULL;
COMMENT ON COLUMN meetings.room_name IS NULL;
COMMENT ON COLUMN meetings.recurring_meeting_id IS NULL;

-- calendar_events
COMMENT ON TABLE calendar_events IS NULL;
COMMENT ON COLUMN calendar_events.rrule IS NULL;
COMMENT ON COLUMN calendar_events.recurrence_end IS NULL;
COMMENT ON COLUMN calendar_events.timezone IS NULL;

-- dialer_campaigns
COMMENT ON TABLE dialer_campaigns IS NULL;
COMMENT ON COLUMN dialer_campaigns.mode IS NULL;
COMMENT ON COLUMN dialer_campaigns.settings IS NULL;
COMMENT ON COLUMN dialer_campaigns.assigned_agent_ids IS NULL;
COMMENT ON COLUMN dialer_campaigns.contact_count IS NULL;

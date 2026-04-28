-- Option-B Phase 1: add tenant_id to 20 core tables
-- Sentinel: 00000000-0000-0000-0000-000000000001 (existing single-tenant data)
DO $$
DECLARE
    t TEXT;
    tables TEXT[] := ARRAY[
        'deals','activities','tasks','projects','channels','messages',
        'notifications','time_entries','calendar_events','email_messages',
        'inbox_messages','deal_stage_history','pipeline_stages',
        'saved_filters','custom_field_definitions','automations',
        'document_files','recordings','dialer_call_sessions','audit_log'
    ];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        EXECUTE format(
            'ALTER TABLE %I ADD COLUMN IF NOT EXISTS tenant_id UUID NOT NULL DEFAULT %L',
            t, '00000000-0000-0000-0000-000000000001'
        );
        EXECUTE format(
            'CREATE INDEX IF NOT EXISTS idx_%I_tenant_id ON %I(tenant_id)',
            t, t
        );
    END LOOP;
END $$;

-- Composite indexes for hot-paths
-- deals: stage_id exists (000009)
CREATE INDEX IF NOT EXISTS idx_deals_tenant_stage ON deals(tenant_id, stage_id);
-- activities: assigned_to exists (000010); column is assigned_to, not owner_id
CREATE INDEX IF NOT EXISTS idx_activities_tenant_assigned ON activities(tenant_id, assigned_to);
-- tasks: status_id exists (000025)
CREATE INDEX IF NOT EXISTS idx_tasks_tenant_status ON tasks(tenant_id, status_id);
-- messages: channel_id exists (000015)
CREATE INDEX IF NOT EXISTS idx_messages_tenant_channel ON messages(tenant_id, channel_id);
-- notifications: user_id exists (000021)
CREATE INDEX IF NOT EXISTS idx_notifications_tenant_user ON notifications(tenant_id, user_id);
-- time_entries: user_id exists (000030)
CREATE INDEX IF NOT EXISTS idx_time_entries_tenant_user ON time_entries(tenant_id, user_id);
-- calendar_events: start_time exists (000033); column is start_time, not start_at
CREATE INDEX IF NOT EXISTS idx_calendar_events_tenant_start ON calendar_events(tenant_id, start_time);
-- recordings: status exists (000037)
CREATE INDEX IF NOT EXISTS idx_recordings_tenant_status ON recordings(tenant_id, status);
-- audit_log: timestamp exists (000039); column is timestamp, not occurred_at
CREATE INDEX IF NOT EXISTS idx_audit_log_tenant_time ON audit_log(tenant_id, timestamp DESC);

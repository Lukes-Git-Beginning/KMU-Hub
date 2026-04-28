-- Rollback Option-B Phase 1: remove tenant_id from 20 core tables
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
        -- Drop per-table tenant_id index first
        EXECUTE format('DROP INDEX IF EXISTS idx_%I_tenant_id', t);
        EXECUTE format('ALTER TABLE %I DROP COLUMN IF EXISTS tenant_id', t);
    END LOOP;
END $$;

-- Drop composite indexes
DROP INDEX IF EXISTS idx_deals_tenant_stage;
DROP INDEX IF EXISTS idx_activities_tenant_assigned;
DROP INDEX IF EXISTS idx_tasks_tenant_status;
DROP INDEX IF EXISTS idx_messages_tenant_channel;
DROP INDEX IF EXISTS idx_notifications_tenant_user;
DROP INDEX IF EXISTS idx_time_entries_tenant_user;
DROP INDEX IF EXISTS idx_calendar_events_tenant_start;
DROP INDEX IF EXISTS idx_recordings_tenant_status;
DROP INDEX IF EXISTS idx_audit_log_tenant_time;

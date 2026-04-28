-- Rollback Option-B Phase 1: remove tenant_id from 20 core tables
--
-- Sentinel-UUID note (P1 #31 / Welle-3.5):
-- The up-migration backfills existing rows with the sentinel
-- '00000000-0000-0000-0000-000000000001'. That sentinel does NOT correspond to a
-- real row in the tenants table, so a FK tenant_id -> tenants(id) is intentionally
-- absent from 000106. The FK will be added after Pilot-1 once legacy rows are
-- purged or re-assigned to real Tenant IDs (planned Sprint 3 cleanup migration).
-- Do NOT add the FK here without first resolving the sentinel mismatch.
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

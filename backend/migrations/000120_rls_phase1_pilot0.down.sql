-- Rollback for 000120: drop policies and disable RLS on the 22 Pilot-0 tables.
-- Helper functions and procedures from 000118 stay in place — they are
-- side-effect free without active policies.

SET LOCAL row_security = off;

BEGIN;

-- Custom-policy tables first (named policies).
DROP POLICY IF EXISTS user_isolation ON users;
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_self_isolation ON tenants;
ALTER TABLE tenants NO FORCE ROW LEVEL SECURITY;
ALTER TABLE tenants DISABLE ROW LEVEL SECURITY;

-- Standard-policy tables (policy name is `tenant_isolation`, set by
-- enable_tenant_rls procedure in 000118).
DO $$
DECLARE
    t text;
    standard_tables text[] := ARRAY[
        'user_sessions',
        'idempotency_keys',
        'audit_log',
        'recording_consents',
        'recordings',
        'consent_records',
        'dialer_agent_status_log',
        'dialer_call_events',
        'dialer_call_outcomes',
        'dialer_call_sessions',
        'dialer_campaign_contacts',
        'dialer_campaigns',
        'tasks',
        'deal_stage_history',
        'pipeline_stages',
        'activities',
        'deals',
        'companies',
        'contacts'
    ];
BEGIN
    FOREACH t IN ARRAY standard_tables LOOP
        EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
        EXECUTE format('ALTER TABLE %I NO FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I DISABLE ROW LEVEL SECURITY', t);
    END LOOP;
END$$;

COMMIT;

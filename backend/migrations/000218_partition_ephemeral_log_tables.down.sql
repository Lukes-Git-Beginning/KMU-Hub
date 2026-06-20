-- R2-P1.10 DOWN — Remove pg_cron jobs, helper functions, and partitioned tables.
-- Restores events, dialer_call_events, automation_executions as plain tables.
--
-- NOTE: This migration copies all data back from the partitioned table into a
-- plain replacement. In production, if partitions have already been dropped by
-- the retention jobs, those rows are lost (by design — they are ephemeral).

BEGIN;

-- =============================================================================
-- 1) Remove pg_cron jobs (exception-safe)
-- =============================================================================

DO $$
BEGIN
    PERFORM cron.unschedule('partition-advance-events');
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron job partition-advance-events not found or pg_cron not available: %', SQLERRM;
END;
$$;

DO $$
BEGIN
    PERFORM cron.unschedule('partition-retention-drop');
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron job partition-retention-drop not found or pg_cron not available: %', SQLERRM;
END;
$$;

-- =============================================================================
-- 2) Restore events as plain table
-- =============================================================================

ALTER TABLE events RENAME TO events_partitioned;

-- Free index/constraint names held by the renamed partitioned table.
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT conname FROM pg_constraint WHERE conrelid = 'events_partitioned'::regclass LOOP
        EXECUTE format('ALTER TABLE events_partitioned DROP CONSTRAINT IF EXISTS %I', r.conname);
    END LOOP;
    FOR r IN SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'events_partitioned' LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I', r.indexname);
    END LOOP;
END;
$$;

CREATE TABLE events (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type_key VARCHAR(100) NOT NULL,
    module_id      VARCHAR(50)  NOT NULL,
    priority       VARCHAR(10)  NOT NULL DEFAULT 'normal'
                       CHECK (priority IN ('urgent', 'normal', 'low')),
    actor_id       UUID,
    resource_id    VARCHAR(255),
    payload        JSONB,
    processed      BOOLEAN      NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_unprocessed ON events(created_at ASC) WHERE processed = false;

INSERT INTO events SELECT * FROM events_partitioned;

DROP TABLE events_partitioned CASCADE;

-- =============================================================================
-- 3) Restore dialer_call_events as plain table
-- =============================================================================

ALTER TABLE dialer_call_events RENAME TO dialer_call_events_partitioned;

-- Free index/constraint names held by the renamed partitioned table.
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT conname FROM pg_constraint WHERE conrelid = 'dialer_call_events_partitioned'::regclass LOOP
        EXECUTE format('ALTER TABLE dialer_call_events_partitioned DROP CONSTRAINT IF EXISTS %I', r.conname);
    END LOOP;
    FOR r IN SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'dialer_call_events_partitioned' LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I', r.indexname);
    END LOOP;
END;
$$;

CREATE TABLE dialer_call_events (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    dialer_call_session_id UUID        NOT NULL
        REFERENCES dialer_call_sessions(id) ON DELETE CASCADE,
    event_type             TEXT        NOT NULL,
    payload                JSONB       NOT NULL DEFAULT '{}',
    occurred_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id              UUID        NOT NULL,
    CONSTRAINT fk_dialer_call_events_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_dce_session
    ON dialer_call_events(dialer_call_session_id, occurred_at);
CREATE INDEX idx_dialer_call_events_tenant_id
    ON dialer_call_events(tenant_id);

INSERT INTO dialer_call_events SELECT * FROM dialer_call_events_partitioned;

CALL enable_tenant_rls('dialer_call_events');

DROP TABLE dialer_call_events_partitioned CASCADE;

-- =============================================================================
-- 4) Restore automation_executions as plain table
-- =============================================================================

ALTER TABLE automation_executions RENAME TO automation_executions_partitioned;

-- Free index/constraint names held by the renamed partitioned table.
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT conname FROM pg_constraint WHERE conrelid = 'automation_executions_partitioned'::regclass LOOP
        EXECUTE format('ALTER TABLE automation_executions_partitioned DROP CONSTRAINT IF EXISTS %I', r.conname);
    END LOOP;
    FOR r IN SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'automation_executions_partitioned' LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I', r.indexname);
    END LOOP;
END;
$$;

CREATE TABLE automation_executions (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    automation_id    UUID        NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    chain_id         UUID        NOT NULL,
    trigger_event    JSONB       NOT NULL,
    condition_result BOOLEAN     NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'running'
                         CHECK (status IN ('running', 'completed', 'failed', 'skipped', 'aborted')),
    steps            JSONB       NOT NULL DEFAULT '[]',
    error_message    TEXT,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ,
    duration_ms      INTEGER,
    tenant_id        UUID        NOT NULL REFERENCES tenants(id)
);

CREATE INDEX idx_automation_executions_automation
    ON automation_executions(automation_id, started_at DESC);
CREATE INDEX idx_automation_executions_status
    ON automation_executions(status, started_at DESC)
    WHERE status IN ('failed', 'running');
CREATE INDEX idx_automation_executions_cleanup
    ON automation_executions(started_at)
    WHERE completed_at IS NOT NULL;
CREATE INDEX idx_automation_executions_tenant_id
    ON automation_executions(tenant_id);

INSERT INTO automation_executions SELECT * FROM automation_executions_partitioned;

CALL enable_tenant_rls('automation_executions');

DROP TABLE automation_executions_partitioned CASCADE;

-- =============================================================================
-- 5) Drop helper functions
-- =============================================================================

DROP FUNCTION IF EXISTS create_monthly_partition(regclass, date);
DROP FUNCTION IF EXISTS drop_old_partitions(text, int);

COMMIT;

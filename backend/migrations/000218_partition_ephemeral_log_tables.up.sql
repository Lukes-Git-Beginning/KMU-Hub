-- R2-P1.10 — Declarative range-partitioning + pg_cron retention for three
-- ephemeral log tables: events, dialer_call_events, automation_executions.
-- audit_log is intentionally excluded (§257/§147 AO statutory retention + hash chain).
--
-- RLS state per table:
--   events                 — NO tenant_id, NO RLS (system-level event bus, mig 000021)
--   dialer_call_events     — tenant_id NOT NULL + FK (mig 000119), RLS active (mig 000120)
--   automation_executions  — tenant_id NOT NULL (mig 000112), RLS active (mig 000122)
--
-- Partition key constraint:
--   Declarative partitioning requires the partition key to be part of every
--   PRIMARY KEY / UNIQUE constraint. PKs are promoted to (id, <time_col>).
--
-- pg_cron safety:
--   All cron statements are wrapped in exception-safe DO blocks so this migration
--   runs cleanly on a Postgres instance without pg_cron preloaded (CI, dev, etc.).

BEGIN;

-- =============================================================================
-- 0) Helper functions for partition management (SECURITY DEFINER)
-- =============================================================================

-- create_monthly_partition: creates a single RANGE partition for one calendar month.
-- Idempotent — uses CREATE TABLE IF NOT EXISTS.
CREATE OR REPLACE FUNCTION create_monthly_partition(
    parent_table regclass,
    month_start  date
) RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    month_end      date := (month_start + interval '1 month')::date;
    partition_name text := format(
        '%s_%s',
        parent_table::text,
        to_char(month_start, 'YYYY_MM')
    );
BEGIN
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF %s FOR VALUES FROM (%L) TO (%L)',
        partition_name,
        parent_table,
        month_start,
        month_end
    );
END;
$$;

-- drop_old_partitions: drops RANGE partitions whose upper bound is older than
-- now() - keep_days. The DEFAULT partition is never dropped.
CREATE OR REPLACE FUNCTION drop_old_partitions(
    parent_table_name text,
    keep_days         int
) RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    rec         record;
    cutoff      timestamptz := now() - (keep_days || ' days')::interval;
    upper_bound timestamptz;
BEGIN
    FOR rec IN
        SELECT
            c.relname                              AS partition_name,
            pg_get_expr(c.relpartbound, c.oid)    AS bound_expr
        FROM pg_class p
        JOIN pg_inherits i ON i.inhparent = p.oid
        JOIN pg_class c    ON c.oid = i.inhrelid
        WHERE p.relname = parent_table_name
          AND c.relkind = 'r'
          -- exclude DEFAULT partition
          AND pg_get_expr(c.relpartbound, c.oid) <> 'DEFAULT'
    LOOP
        -- Extract the upper bound from the partition bound expression.
        -- Bound looks like: FOR VALUES FROM ('2025-01-01') TO ('2025-02-01')
        SELECT (regexp_match(rec.bound_expr, 'TO \(''([^'']+)''\)'))[1]::timestamptz
        INTO upper_bound;

        IF upper_bound IS NOT NULL AND upper_bound <= cutoff THEN
            EXECUTE format('DROP TABLE IF EXISTS %I', rec.partition_name);
            RAISE NOTICE 'Dropped old partition: %', rec.partition_name;
        END IF;
    END LOOP;
END;
$$;

-- =============================================================================
-- 1) events  (mig 000021: no tenant_id, no RLS — system-level event bus)
-- =============================================================================

ALTER TABLE events RENAME TO events_old;

-- RENAME TO does not rename dependent objects (indexes, constraints) in Postgres,
-- so their canonical names stay bound to events_old and would collide with the new
-- table. Drop them on the old table (not needed for the SELECT * data copy) to free
-- the names.
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT conname FROM pg_constraint WHERE conrelid = 'events_old'::regclass LOOP
        EXECUTE format('ALTER TABLE events_old DROP CONSTRAINT IF EXISTS %I', r.conname);
    END LOOP;
    FOR r IN SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'events_old' LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I', r.indexname);
    END LOOP;
END;
$$;

CREATE TABLE events (
    id             UUID         NOT NULL DEFAULT gen_random_uuid(),
    event_type_key VARCHAR(100) NOT NULL,
    module_id      VARCHAR(50)  NOT NULL,
    priority       VARCHAR(10)  NOT NULL DEFAULT 'normal'
                       CHECK (priority IN ('urgent', 'normal', 'low')),
    actor_id       UUID,
    resource_id    VARCHAR(255),
    payload        JSONB,
    processed      BOOLEAN      NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Index on parent propagates to all partitions
CREATE INDEX idx_events_unprocessed ON events(created_at ASC) WHERE processed = false;

-- Seed monthly partitions: 3 months back .. 12 months forward from migration date
DO $$
DECLARE
    m date;
BEGIN
    FOR m IN
        SELECT generate_series(
            date_trunc('month', now()) - interval '3 months',
            date_trunc('month', now()) + interval '12 months',
            interval '1 month'
        )::date
    LOOP
        PERFORM create_monthly_partition('events'::regclass, m);
    END LOOP;
END;
$$;

-- DEFAULT catch-all for any rows outside the seeded window
CREATE TABLE events_default PARTITION OF events DEFAULT;

INSERT INTO events SELECT * FROM events_old;

DROP TABLE events_old;

-- =============================================================================
-- 2) dialer_call_events  (tenant_id NOT NULL, RLS active)
--    Original columns: id, dialer_call_session_id, event_type, payload,
--                      occurred_at, tenant_id (mig 000119)
-- =============================================================================

ALTER TABLE dialer_call_events RENAME TO dialer_call_events_old;

-- Free index/constraint names held by the renamed old table.
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT conname FROM pg_constraint WHERE conrelid = 'dialer_call_events_old'::regclass LOOP
        EXECUTE format('ALTER TABLE dialer_call_events_old DROP CONSTRAINT IF EXISTS %I', r.conname);
    END LOOP;
    FOR r IN SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'dialer_call_events_old' LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I', r.indexname);
    END LOOP;
END;
$$;

CREATE TABLE dialer_call_events (
    id                     UUID        NOT NULL DEFAULT gen_random_uuid(),
    dialer_call_session_id UUID        NOT NULL,
    event_type             TEXT        NOT NULL,
    payload                JSONB       NOT NULL DEFAULT '{}',
    occurred_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tenant_id              UUID        NOT NULL,
    PRIMARY KEY (id, occurred_at),
    CONSTRAINT fk_dce_session
        FOREIGN KEY (dialer_call_session_id)
        REFERENCES dialer_call_sessions(id) ON DELETE CASCADE,
    CONSTRAINT fk_dialer_call_events_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) PARTITION BY RANGE (occurred_at);

CREATE INDEX idx_dce_session
    ON dialer_call_events(dialer_call_session_id, occurred_at);
CREATE INDEX idx_dialer_call_events_tenant_id
    ON dialer_call_events(tenant_id);

-- Seed monthly partitions
DO $$
DECLARE
    m date;
BEGIN
    FOR m IN
        SELECT generate_series(
            date_trunc('month', now()) - interval '3 months',
            date_trunc('month', now()) + interval '12 months',
            interval '1 month'
        )::date
    LOOP
        PERFORM create_monthly_partition('dialer_call_events'::regclass, m);
    END LOOP;
END;
$$;

CREATE TABLE dialer_call_events_default PARTITION OF dialer_call_events DEFAULT;

INSERT INTO dialer_call_events SELECT * FROM dialer_call_events_old;

-- Reactivate RLS on new partitioned parent — policies propagate to all partitions
CALL enable_tenant_rls('dialer_call_events');

DROP TABLE dialer_call_events_old;

-- =============================================================================
-- 3) automation_executions  (tenant_id NOT NULL, RLS active)
--    Original columns: id, automation_id, chain_id, trigger_event,
--                      condition_result, status, steps, error_message,
--                      started_at, completed_at, duration_ms,
--                      tenant_id (mig 000112)
-- =============================================================================

ALTER TABLE automation_executions RENAME TO automation_executions_old;

-- Free index/constraint names held by the renamed old table.
DO $$
DECLARE r record;
BEGIN
    FOR r IN SELECT conname FROM pg_constraint WHERE conrelid = 'automation_executions_old'::regclass LOOP
        EXECUTE format('ALTER TABLE automation_executions_old DROP CONSTRAINT IF EXISTS %I', r.conname);
    END LOOP;
    FOR r IN SELECT indexname FROM pg_indexes WHERE schemaname = 'public' AND tablename = 'automation_executions_old' LOOP
        EXECUTE format('DROP INDEX IF EXISTS %I', r.indexname);
    END LOOP;
END;
$$;

CREATE TABLE automation_executions (
    id               UUID        NOT NULL DEFAULT gen_random_uuid(),
    automation_id    UUID        NOT NULL,
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
    tenant_id        UUID        NOT NULL,
    PRIMARY KEY (id, started_at),
    CONSTRAINT fk_ae_automation
        FOREIGN KEY (automation_id) REFERENCES automations(id) ON DELETE CASCADE,
    CONSTRAINT fk_automation_executions_tenant
        FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) PARTITION BY RANGE (started_at);

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

-- Seed monthly partitions
DO $$
DECLARE
    m date;
BEGIN
    FOR m IN
        SELECT generate_series(
            date_trunc('month', now()) - interval '3 months',
            date_trunc('month', now()) + interval '12 months',
            interval '1 month'
        )::date
    LOOP
        PERFORM create_monthly_partition('automation_executions'::regclass, m);
    END LOOP;
END;
$$;

CREATE TABLE automation_executions_default PARTITION OF automation_executions DEFAULT;

INSERT INTO automation_executions SELECT * FROM automation_executions_old;

CALL enable_tenant_rls('automation_executions');

DROP TABLE automation_executions_old;

-- =============================================================================
-- 4) pg_cron extension + scheduled jobs (all exception-safe for CI-safety)
-- =============================================================================

-- Extension (requires pg_cron in shared_preload_libraries — see Dockerfile)
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_cron;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron not available (shared_preload_libraries not set or not superuser) — skipping. Install pg_cron and rerun, or schedule partition jobs manually.';
END;
$$;

-- Job 1: Monthly — pre-create partitions 1 and 2 months ahead for all three tables.
-- Runs on the 25th of every month at 02:00 UTC.
DO $$
BEGIN
    PERFORM cron.schedule(
        'partition-advance-events',
        '0 2 25 * *',
        $cmd$
        SELECT create_monthly_partition('events'::regclass,
               (date_trunc('month', now()) + interval '1 month')::date);
        SELECT create_monthly_partition('events'::regclass,
               (date_trunc('month', now()) + interval '2 months')::date);
        SELECT create_monthly_partition('dialer_call_events'::regclass,
               (date_trunc('month', now()) + interval '1 month')::date);
        SELECT create_monthly_partition('dialer_call_events'::regclass,
               (date_trunc('month', now()) + interval '2 months')::date);
        SELECT create_monthly_partition('automation_executions'::regclass,
               (date_trunc('month', now()) + interval '1 month')::date);
        SELECT create_monthly_partition('automation_executions'::regclass,
               (date_trunc('month', now()) + interval '2 months')::date);
        $cmd$
    );
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron schedule (partition-advance-events) skipped: %. Register manually after pg_cron is available.', SQLERRM;
END;
$$;

-- Job 2: Daily — drop partitions older than 90-day retention window.
-- Runs every day at 03:30 UTC.
DO $$
BEGIN
    PERFORM cron.schedule(
        'partition-retention-drop',
        '30 3 * * *',
        $cmd$
        SELECT drop_old_partitions('events', 90);
        SELECT drop_old_partitions('dialer_call_events', 90);
        SELECT drop_old_partitions('automation_executions', 90);
        $cmd$
    );
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_cron schedule (partition-retention-drop) skipped: %. Register manually after pg_cron is available.', SQLERRM;
END;
$$;

COMMIT;

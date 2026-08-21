-- Migration: 000316_create_retention_runs
-- DSGVO Art. 5(1)(e) — run log for the retention engine.
--
-- `retention_policies` (000233) could be maintained since day one, but nothing
-- ever executed one. These two tables are the record of what an execution
-- actually did: one row per run, one row per policy inside that run. Without
-- them a retention policy remains an unverifiable claim.
--
-- Same RLS pattern as retention_policies itself (enable_tenant_rls, 000118).

CREATE TABLE retention_runs (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    -- dry_run counts only and is the default; enforce is the armed mode.
    mode             VARCHAR(16) NOT NULL CHECK (mode IN ('dry_run', 'enforce')),
    status           VARCHAR(16) NOT NULL CHECK (status IN ('running', 'completed', 'failed')),
    -- Free label of what started the run ('manual', 'schedule', ...), so an
    -- unexpected deletion can be traced back to its trigger.
    triggered_by     VARCHAR(32) NOT NULL DEFAULT 'manual',
    policies_total   INT         NOT NULL DEFAULT 0,
    records_matched  INT         NOT NULL DEFAULT 0,
    records_affected INT         NOT NULL DEFAULT 0,
    records_skipped  INT         NOT NULL DEFAULT 0,
    error            TEXT,
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ
);

CREATE INDEX idx_retention_runs_tenant_started ON retention_runs (tenant_id, started_at DESC);

CREATE TABLE retention_run_items (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID        NOT NULL,
    run_id         UUID        NOT NULL REFERENCES retention_runs(id) ON DELETE CASCADE,
    -- The policy may be edited or removed after the run; the item keeps the
    -- values it actually ran with, so the log stays readable either way.
    policy_id      UUID        REFERENCES retention_policies(id) ON DELETE SET NULL,
    resource_type  VARCHAR(64) NOT NULL,
    action         VARCHAR(16) NOT NULL,
    retention_days INT         NOT NULL,
    cutoff         TIMESTAMPTZ NOT NULL,
    -- unmapped: no handler is registered for this resource_type. The policy
    -- looks fulfilled in the UI but deletes nothing -- it must stay visible.
    -- unsupported: a handler exists but cannot perform the policy's action.
    status         VARCHAR(24) NOT NULL CHECK (status IN ('dry_run', 'applied', 'unmapped', 'unsupported', 'failed')),
    matched        INT         NOT NULL DEFAULT 0,
    affected       INT         NOT NULL DEFAULT 0,
    skipped        INT         NOT NULL DEFAULT 0,
    -- [{record_id, reason}], truncated -- see retentionMaxSkipReasons in Go.
    skip_reasons   JSONB       NOT NULL DEFAULT '[]'::jsonb,
    message        TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_retention_run_items_run ON retention_run_items (run_id);
CREATE INDEX idx_retention_run_items_tenant_resource ON retention_run_items (tenant_id, resource_type);

CALL enable_tenant_rls('retention_runs');
CALL enable_tenant_rls('retention_run_items');

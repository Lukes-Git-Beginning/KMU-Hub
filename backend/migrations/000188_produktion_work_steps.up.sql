-- 000188 Produktion Arbeitsschritte (Work Steps)

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS production_work_steps (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    order_id        UUID        NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
    step_nr         INTEGER     NOT NULL DEFAULT 1,
    name            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    duration_minutes INTEGER    NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','in_progress','completed','skipped')),
    assignee        TEXT        NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_production_work_steps_order
    ON production_work_steps (order_id, step_nr);
CREATE INDEX IF NOT EXISTS idx_production_work_steps_tenant
    ON production_work_steps (tenant_id);

CALL enable_tenant_rls('production_work_steps');

COMMIT;

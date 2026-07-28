-- Migration 000246: recurring invoice schedules (Abo-Rechnungen)
--
-- A recurring invoice is a template that emits draft invoices at a fixed
-- interval. finance-client.ts already calls /api/v1/finance/recurring{,/:id,
-- /:id/pause,/resume,/generate}; until now nothing served it.
--
-- Generation must be idempotent: a double click, a retried request or a second
-- scheduler pass must never emit the same period twice — a duplicated invoice
-- is a real business event, not a display glitch. finance_recurring_runs is the
-- idempotency ledger: one row per (schedule, period), unique per tenant, with
-- the emitted invoice id. A replay finds the row and returns the invoice that
-- already exists instead of creating a second one.
--
-- The column is named recurrence_interval, not interval: INTERVAL is a type
-- keyword in PostgreSQL and would need quoting in every statement.

BEGIN;

CREATE TABLE finance_recurring_invoices (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    title               TEXT        NOT NULL,
    customer_name       TEXT        NOT NULL DEFAULT '',
    customer_address    TEXT        NOT NULL DEFAULT '',
    customer_email      TEXT        NOT NULL DEFAULT '',
    customer_ust_id_nr  TEXT        NOT NULL DEFAULT '',
    tax_mode            TEXT        NOT NULL DEFAULT 'standard',
    -- Same JSONB shapes as finance_invoices.line_items / .tax_breakdown, so the
    -- generated invoice is a straight copy of the schedule's template.
    line_items          JSONB       NOT NULL DEFAULT '[]'::jsonb,
    tax_breakdown       JSONB       NOT NULL DEFAULT '{}'::jsonb,
    currency            TEXT        NOT NULL DEFAULT 'EUR',
    recurrence_interval TEXT        NOT NULL,
    status              TEXT        NOT NULL DEFAULT 'active',
    start_date          DATE        NOT NULL,
    end_date            DATE        NULL,
    next_run            DATE        NOT NULL,
    payment_terms_days  INTEGER     NOT NULL DEFAULT 30,
    generated_count     INTEGER     NOT NULL DEFAULT 0,
    last_generated_at   DATE        NULL,
    notes               TEXT        NOT NULL DEFAULT '',
    created_by          UUID        NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT finance_recurring_invoices_interval_check
        CHECK (recurrence_interval IN ('weekly', 'monthly', 'quarterly', 'yearly')),
    CONSTRAINT finance_recurring_invoices_status_check
        CHECK (status IN ('active', 'paused', 'ended')),
    CONSTRAINT finance_recurring_invoices_terms_check
        CHECK (payment_terms_days >= 0),
    CONSTRAINT finance_recurring_invoices_end_after_start_check
        CHECK (end_date IS NULL OR end_date >= start_date)
);

-- List view: tenant-scoped, newest first.
CREATE INDEX idx_finance_recurring_tenant_created
    ON finance_recurring_invoices (tenant_id, created_at DESC);
-- Due-schedule scan for a future scheduler pass.
CREATE INDEX idx_finance_recurring_tenant_due
    ON finance_recurring_invoices (tenant_id, next_run)
    WHERE status = 'active';

ALTER TABLE finance_recurring_invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance_recurring_invoices FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON finance_recurring_invoices
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

-- Idempotency ledger. period_date is the next_run value that was consumed, so
-- the unique constraint is what actually blocks a second emission — not an
-- application-side check that a concurrent request could race past.
CREATE TABLE finance_recurring_runs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    recurring_id UUID        NOT NULL REFERENCES finance_recurring_invoices (id) ON DELETE CASCADE,
    period_date  DATE        NOT NULL,
    invoice_id   UUID        NULL REFERENCES finance_invoices (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT finance_recurring_runs_period_unique
        UNIQUE (tenant_id, recurring_id, period_date)
);

CREATE INDEX idx_finance_recurring_runs_schedule
    ON finance_recurring_runs (tenant_id, recurring_id, period_date DESC);

ALTER TABLE finance_recurring_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance_recurring_runs FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON finance_recurring_runs
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

-- Back-link on the emitted invoice: the frontend lists a schedule's invoices via
-- GET /finance/invoices?recurring_id=..., and the invoice detail shows its origin.
ALTER TABLE finance_invoices
    ADD COLUMN recurring_id UUID NULL REFERENCES finance_recurring_invoices (id) ON DELETE SET NULL;

CREATE INDEX idx_finance_invoices_recurring
    ON finance_invoices (tenant_id, recurring_id)
    WHERE recurring_id IS NOT NULL;

COMMIT;

-- Migration 000257: expenses (Ausgaben) with a two-person approval.
--
-- An expense is a claim one employee makes and another one decides. The schema
-- carries that as two distinct actors: submitted_by is who entered it,
-- decided_by is who approved or rejected it. They are kept apart because the
-- service refuses a decision by the submitter -- a rule that only holds if the
-- two identities are actually stored separately, not derived from each other.
--
-- status is a CHECK rather than an enum type so the set can be widened by a
-- later migration without an ALTER TYPE; the three values match the frontend's
-- Expense.status union (stores/finance.ts) exactly.
--
-- account is the SKR03/04 Sachkonto for the DATEV export. It is nullable on
-- purpose: an expense is entered by whoever paid it, and the bookkeeping
-- account is assigned afterwards ("Kontieren"). Forcing it at insert time would
-- push the accountant's job onto the person holding the receipt.
--
-- receipt_name holds the filename of the attached document. There is no blob
-- here: the frontend's receipt flow (finance-client.ts financeExpenseApi.
-- attachReceipt) posts a filename, not bytes. See the lean: note in
-- internal/biz/expense/service.go for when that grows into a real upload.

BEGIN;

CREATE TABLE finance_expenses (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID          NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    description  TEXT          NOT NULL,
    amount       NUMERIC(15,2) NOT NULL,
    expense_date DATE          NOT NULL,
    category     TEXT          NOT NULL DEFAULT '',
    supplier     TEXT          NOT NULL DEFAULT '',
    project      TEXT          NULL,
    -- SKR03/04 Sachkonto, assigned during Kontierung.
    account      TEXT          NULL,
    receipt_name TEXT          NULL,
    status       TEXT          NOT NULL DEFAULT 'pending',
    submitted_by UUID          NULL REFERENCES users (id) ON DELETE SET NULL,
    decided_by   UUID          NULL REFERENCES users (id) ON DELETE SET NULL,
    decided_at   TIMESTAMPTZ   NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT finance_expenses_status_check
        CHECK (status IN ('pending', 'approved', 'rejected')),
    CONSTRAINT finance_expenses_amount_positive
        CHECK (amount > 0)
);

-- The list view is "this tenant, newest first"; the approval queue filters on
-- status. Both are covered without a sequential scan.
CREATE INDEX idx_finance_expenses_tenant_date
    ON finance_expenses (tenant_id, expense_date DESC, created_at DESC);
CREATE INDEX idx_finance_expenses_tenant_status
    ON finance_expenses (tenant_id, status);
-- Supports the "own" data scope on the list endpoint (RBAC Phase 2).
CREATE INDEX idx_finance_expenses_tenant_submitter
    ON finance_expenses (tenant_id, submitted_by);

ALTER TABLE finance_expenses ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance_expenses FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON finance_expenses
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;

-- Migration 000247: bank statement import (Zahlungsabgleich CAMT.053 / MT940)
--
-- A bank statement is a file a bank hands out, not data this system authored.
-- Two consequences shape this schema.
--
-- First, idempotency. The same export gets downloaded and uploaded twice more
-- often than not — a second browser tab, a retried request, a colleague doing
-- the same job. Re-importing must not duplicate transactions, because a
-- duplicated bank entry either double-matches an invoice or leaves a phantom
-- receivable behind. content_hash is the SHA-256 of the raw uploaded bytes and
-- is unique per tenant: a replay finds the existing statement and returns it
-- instead of writing a second one. The constraint, not an application-side
-- lookup, is what blocks it — two concurrent uploads would race past a check.
--
-- Second, the transactions are a record of what the bank reported, so they are
-- kept verbatim (amount, dates, remittance text) and the reconciliation state
-- is carried in separate columns. Matching never rewrites what was imported.

BEGIN;

CREATE TABLE finance_bank_statements (
    id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID          NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    format            TEXT          NOT NULL,
    filename          TEXT          NOT NULL DEFAULT '',
    -- SHA-256 (hex) of the raw file. Idempotency key of the whole import.
    content_hash      TEXT          NOT NULL,
    -- Statement header, as far as the format carries it.
    account_iban      TEXT          NOT NULL DEFAULT '',
    statement_ref     TEXT          NOT NULL DEFAULT '',
    currency          TEXT          NOT NULL DEFAULT 'EUR',
    opening_balance   NUMERIC(15,2) NULL,
    closing_balance   NUMERIC(15,2) NULL,
    statement_date    DATE          NULL,
    transaction_count INTEGER       NOT NULL DEFAULT 0,
    imported_by       UUID          NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT finance_bank_statements_format_check
        CHECK (format IN ('camt053', 'mt940')),
    CONSTRAINT finance_bank_statements_hash_unique
        UNIQUE (tenant_id, content_hash)
);

CREATE INDEX idx_finance_bank_statements_tenant_created
    ON finance_bank_statements (tenant_id, created_at DESC);

ALTER TABLE finance_bank_statements ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance_bank_statements FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON finance_bank_statements
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

CREATE TABLE finance_bank_transactions (
    id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID          NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    statement_id      UUID          NOT NULL REFERENCES finance_bank_statements (id) ON DELETE CASCADE,
    -- What the bank reported. Never rewritten by reconciliation.
    entry_ref         TEXT          NOT NULL DEFAULT '',
    end_to_end_id     TEXT          NOT NULL DEFAULT '',
    value_date        DATE          NOT NULL,
    booking_date      DATE          NULL,
    -- Signed: a credit (money in) is positive, a debit negative. Storing the
    -- sign rather than a separate direction flag keeps every sum a plain SUM().
    amount            NUMERIC(15,2) NOT NULL,
    currency          TEXT          NOT NULL DEFAULT 'EUR',
    counterparty_name TEXT          NOT NULL DEFAULT '',
    counterparty_iban TEXT          NOT NULL DEFAULT '',
    remittance_info   TEXT          NOT NULL DEFAULT '',
    -- Reconciliation state, written by the matcher and by the user.
    match_status      TEXT          NOT NULL DEFAULT 'unmatched',
    match_reason      TEXT          NOT NULL DEFAULT '',
    matched_invoice_id UUID         NULL REFERENCES finance_invoices (id) ON DELETE SET NULL,
    payment_id        UUID          NULL REFERENCES finance_payments (id) ON DELETE SET NULL,
    reconciled_at     TIMESTAMPTZ   NULL,
    reconciled_by     UUID          NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT finance_bank_transactions_match_status_check
        CHECK (match_status IN ('unmatched', 'suggested', 'matched', 'ignored')),
    -- A booked transaction must say which invoice it was booked against.
    CONSTRAINT finance_bank_transactions_matched_needs_invoice_check
        CHECK (match_status <> 'matched' OR matched_invoice_id IS NOT NULL)
);

-- List view: the transactions of one statement, oldest entry first.
CREATE INDEX idx_finance_bank_transactions_statement
    ON finance_bank_transactions (tenant_id, statement_id, value_date, created_at);
-- The work queue: everything still waiting for a decision.
CREATE INDEX idx_finance_bank_transactions_open
    ON finance_bank_transactions (tenant_id, value_date DESC)
    WHERE match_status IN ('unmatched', 'suggested');
-- Reverse lookup from an invoice to the bank entries booked against it.
CREATE INDEX idx_finance_bank_transactions_invoice
    ON finance_bank_transactions (tenant_id, matched_invoice_id)
    WHERE matched_invoice_id IS NOT NULL;

ALTER TABLE finance_bank_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance_bank_transactions FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON finance_bank_transactions
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;

-- Migration 000258: bank accounts (Bankkonten-Stammdaten).
--
-- Migration 000247 already stores what a bank *reports* -- statements and their
-- transactions -- but it has no notion of which accounts a tenant actually
-- holds: finance_bank_statements.account_iban is free text copied out of the
-- uploaded file. This table is the master record those imports attach to.
--
-- The IBAN is stored canonically (upper-case, no spaces) because it is the key
-- two things hang off: the unique constraint below, and the join back to the
-- statements of that account. A human types "DE89 3704 0044 0532 0130 00" and a
-- CAMT.053 file carries "DE89370400440532013000"; if both forms could land here
-- the same account would exist twice and its balance would be split across the
-- copies. Grouping for display happens on the way out (dachfmt.FormatIBAN).
--
-- There is deliberately no balance column. A stored balance is only true at the
-- moment it is written, and nothing here writes it afterwards -- the number
-- would silently age into a lie the accountant cannot tell from a real one. The
-- balance is instead read from the closing balance of the newest imported
-- statement for that IBAN, which is the only figure this system actually knows.
-- last_sync follows the same rule: it is when statement data last arrived, not
-- when somebody clicked "connect".
--
-- connected is a master-data flag, not a live session: the PSD2/FinAPI flow is
-- simulated in the desktop client (BankConnectDialog.tsx) and the real
-- integration is P5. connected_at records when it was set so the flag can be
-- told apart from one that was never touched.

BEGIN;

CREATE TABLE finance_bank_accounts (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    bank_name    TEXT        NOT NULL,
    -- Canonical IBAN: upper-case, no separators. See the header.
    iban         TEXT        NOT NULL,
    bic          TEXT        NOT NULL DEFAULT '',
    currency     TEXT        NOT NULL DEFAULT 'EUR',
    connected    BOOLEAN     NOT NULL DEFAULT FALSE,
    connected_at TIMESTAMPTZ NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT finance_bank_accounts_iban_unique
        UNIQUE (tenant_id, iban),
    CONSTRAINT finance_bank_accounts_iban_canonical
        CHECK (iban = UPPER(iban) AND iban !~ '[^A-Z0-9]' AND char_length(iban) BETWEEN 15 AND 34),
    CONSTRAINT finance_bank_accounts_currency_check
        CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT finance_bank_accounts_bank_name_check
        CHECK (bank_name <> '')
);

-- The list view is "this tenant, by bank name". The IBAN tail keeps two
-- accounts at the same bank in a stable order.
CREATE INDEX idx_finance_bank_accounts_tenant_name
    ON finance_bank_accounts (tenant_id, bank_name, iban);

ALTER TABLE finance_bank_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE finance_bank_accounts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON finance_bank_accounts
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

-- Serves the balance/last-sync lookup: newest statement per account IBAN.
-- Without it every account row in the list would scan the statements table.
CREATE INDEX idx_finance_bank_statements_tenant_iban
    ON finance_bank_statements (tenant_id, account_iban, statement_date DESC, created_at DESC);

COMMIT;

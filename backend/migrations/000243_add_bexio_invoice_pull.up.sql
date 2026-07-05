-- Welle 3b: Bexio Invoice-Pull (read-only mirror).
-- Adds provenance to finance_invoices and pull-cursor columns to bexio_sync_configs.
-- Purely additive scaffolding; no active code path until the puller is wired later.

-- Provenance on finance_invoices: 'cosmi' = Cosmi-issued (GoBD number space,
-- RE-YYYY-NNNN), 'bexio' = pulled read-only mirror that keeps its own Bexio number
-- and never enters finance_number_sequences.
ALTER TABLE finance_invoices
    ADD COLUMN source          VARCHAR(20) NOT NULL DEFAULT 'cosmi'
        CHECK (source IN ('cosmi', 'bexio')),
    ADD COLUMN external_id     TEXT,   -- Bexio kb_invoice.id
    ADD COLUMN external_number TEXT;   -- original Bexio document_nr (shown to the user)

-- Dedup / idempotency for repeated pulls of the same Bexio invoice.
-- Vorbild: finance_payments.idempotency_key partial-unique index (000215).
CREATE UNIQUE INDEX idx_finance_invoices_external
    ON finance_invoices (tenant_id, source, external_id)
    WHERE external_id IS NOT NULL;

-- Pull cursor + toggle on the per-tenant Bexio sync config, mirroring the existing
-- contact_sync / payment_poll columns (000055). Default OFF: no pull until enabled.
ALTER TABLE bexio_sync_configs
    ADD COLUMN invoice_pull_enabled          BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN invoice_pull_interval_minutes INTEGER NOT NULL DEFAULT 15,
    ADD COLUMN last_invoice_pull_at          TIMESTAMPTZ;

-- Allow 'invoice_pull' in the sync-log audit trail. Postgres CHECK constraints are
-- immutable, so drop the auto-named inline constraint and re-add the widened one.
ALTER TABLE bexio_sync_log DROP CONSTRAINT IF EXISTS bexio_sync_log_sync_type_check;
ALTER TABLE bexio_sync_log ADD CONSTRAINT bexio_sync_log_sync_type_check
    CHECK (sync_type IN ('contact_full', 'contact_delta', 'invoice_push',
                         'quote_push', 'payment_poll', 'invoice_pull'));

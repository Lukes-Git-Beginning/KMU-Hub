-- Revert Welle 3b invoice-pull scaffolding.

-- Restore the original sync-log CHECK (without 'invoice_pull').
ALTER TABLE bexio_sync_log DROP CONSTRAINT IF EXISTS bexio_sync_log_sync_type_check;
ALTER TABLE bexio_sync_log ADD CONSTRAINT bexio_sync_log_sync_type_check
    CHECK (sync_type IN ('contact_full', 'contact_delta', 'invoice_push',
                         'quote_push', 'payment_poll'));

ALTER TABLE bexio_sync_configs
    DROP COLUMN IF EXISTS invoice_pull_enabled,
    DROP COLUMN IF EXISTS invoice_pull_interval_minutes,
    DROP COLUMN IF EXISTS last_invoice_pull_at;

DROP INDEX IF EXISTS idx_finance_invoices_external;

ALTER TABLE finance_invoices
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS external_id,
    DROP COLUMN IF EXISTS external_number;

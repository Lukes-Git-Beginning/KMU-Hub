-- Migration 000319: track invoiced work time entries to prevent double billing
--
-- CreateInvoiceFromTimeEntries aggregated billable entries with no record of
-- whether an earlier call had already billed them. Two calls for the same
-- employee/period (double-click, retry after timeout, two staff members
-- working the same invoice) produced two separate invoices over the same
-- hours. billed_at is the exclusion filter; invoice_id is best-effort
-- traceability set after the invoice is confirmed created.

ALTER TABLE hr_work_time_entries
    ADD COLUMN IF NOT EXISTS billed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS invoice_id UUID NULL REFERENCES finance_invoices(id) ON DELETE SET NULL;

-- Speeds up the reservation query's WHERE ... AND billed_at IS NULL filter,
-- which runs under FOR UPDATE on every invoice-from-time-entries call.
CREATE INDEX IF NOT EXISTS idx_hr_work_time_entries_unbilled
    ON hr_work_time_entries (tenant_id, employee_id, clock_in)
    WHERE billed_at IS NULL;

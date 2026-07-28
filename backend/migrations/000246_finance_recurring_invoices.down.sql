-- Rollback 000246: drop recurring invoice schedules and their run ledger.

BEGIN;

DROP INDEX IF EXISTS idx_finance_invoices_recurring;
ALTER TABLE finance_invoices DROP COLUMN IF EXISTS recurring_id;

DROP TABLE IF EXISTS finance_recurring_runs;
DROP TABLE IF EXISTS finance_recurring_invoices;

COMMIT;

DROP INDEX IF EXISTS idx_finance_payments_idem;
ALTER TABLE finance_payments DROP COLUMN IF EXISTS idempotency_key;

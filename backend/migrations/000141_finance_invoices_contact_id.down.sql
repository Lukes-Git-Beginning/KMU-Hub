BEGIN;

SET LOCAL row_security = off;

DROP INDEX IF EXISTS idx_finance_invoices_contact;

ALTER TABLE finance_invoices DROP COLUMN IF EXISTS contact_id;

COMMIT;

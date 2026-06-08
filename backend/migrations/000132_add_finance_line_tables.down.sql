-- Reverse of 000132 — drop RLS policies, the three line tables, and the
-- GoBD lock columns. The line_items JSONB columns were never touched by the
-- up migration, so nothing to restore there.

BEGIN;

SET LOCAL row_security = off;

DROP POLICY IF EXISTS tenant_isolation ON finance_credit_note_lines;
DROP POLICY IF EXISTS tenant_isolation ON finance_quote_lines;
DROP POLICY IF EXISTS tenant_isolation ON finance_invoice_lines;

DROP TABLE IF EXISTS finance_credit_note_lines;
DROP TABLE IF EXISTS finance_quote_lines;
DROP TABLE IF EXISTS finance_invoice_lines;

ALTER TABLE finance_invoices
    DROP COLUMN IF EXISTS locked_by,
    DROP COLUMN IF EXISTS locked_at;

COMMIT;

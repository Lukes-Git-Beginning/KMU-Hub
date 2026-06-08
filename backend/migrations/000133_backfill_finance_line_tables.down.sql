-- Reverse of 000133 — remove backfilled rows and clear the lock columns.
-- The source data (line_items JSONB, snapshot_data lock marker) is untouched
-- by the up migration, so the 000133 up can be re-applied to restore.
--
-- Note: deletes ALL rows from the line tables. At down time the relational
-- cutover (Welle 2+) is assumed not yet live in this environment; the line
-- tables hold only backfilled rows. 000132 down drops the tables afterwards.

BEGIN;

SET LOCAL row_security = off;

DELETE FROM finance_credit_note_lines;
DELETE FROM finance_quote_lines;
DELETE FROM finance_invoice_lines;

UPDATE finance_invoices
SET locked_at = NULL,
    locked_by = NULL
WHERE locked_at IS NOT NULL;

COMMIT;

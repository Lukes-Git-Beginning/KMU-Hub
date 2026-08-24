-- Restores the non-partial index from 000045.
--
-- This direction CAN fail: once the partial index has been live, a tenant may
-- legitimately hold several drafts at invoice_number = '', and those rows
-- violate the non-partial index. Delete or send the surplus drafts first --
--   SELECT tenant_id, count(*) FROM finance_invoices
--    WHERE invoice_number = '' GROUP BY 1 HAVING count(*) > 1;
-- -- rather than weakening the index into a non-unique one, which would drop
-- the guarantee that assigned numbers stay unique per tenant.
DROP INDEX IF EXISTS idx_finance_invoices_number;

CREATE UNIQUE INDEX idx_finance_invoices_number
    ON finance_invoices (tenant_id, invoice_number);

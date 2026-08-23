-- fix-invoice-number-unique-index-blocks-second-draft
--
-- Problem: finance_invoices.invoice_number is VARCHAR(30) NOT NULL DEFAULT ''
-- (000045, line 75), so a draft carries the empty string -- not NULL -- until
-- invoice.Service.Send assigns a real number. The unique index created in the
-- same migration (line 184) is NOT partial, so those empty strings compete for
-- uniqueness with each other: a tenant can hold at most ONE unsent draft at any
-- time. The second Create -- manual, from a quote conversion, or from a second
-- recurring.Service.Generate run for a still-open period -- fails with a raw
-- "duplicate key value violates unique constraint idx_finance_invoices_number"
-- (SQLSTATE 23505), surfacing as a 500 rather than as anything a user can act on.
--
-- The index is meant to keep ASSIGNED numbers unique per tenant (GoBD: no number
-- twice), not to limit how many drafts exist. Making it partial expresses exactly
-- that: any number of rows may sit at '', every real number stays unique.
--
-- Relaxing an existing unique index can never fail on existing data -- every row
-- set that satisfied the old index satisfies the new one too -- so this migration
-- is safe to apply without a prior data audit. The reverse is not true; see the
-- .down.sql.
DROP INDEX IF EXISTS idx_finance_invoices_number;

CREATE UNIQUE INDEX idx_finance_invoices_number
    ON finance_invoices (tenant_id, invoice_number)
    WHERE invoice_number <> '';

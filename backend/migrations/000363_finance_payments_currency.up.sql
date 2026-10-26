-- Payments carry their own currency (decision 2026-10-25, variant a). Until now a
-- payment inherited the invoice currency implicitly, so the open-items and
-- paid-amount arithmetic trusted that every payment was booked in invoice currency.
-- Existing rows are backfilled from their invoice (FK fk_finance_payments_invoice
-- guarantees one exists), then the column becomes NOT NULL without a default: a
-- writer that forgets the currency must fail, not silently book EUR. No exchange
-- rate is stored (decision 2026-10-24); the service rejects a currency that differs
-- from the invoice. Plain column addition, RLS policy unchanged.
ALTER TABLE finance_payments ADD COLUMN currency CHAR(3);

UPDATE finance_payments p
SET currency = i.currency
FROM finance_invoices i
WHERE i.id = p.invoice_id AND i.tenant_id = p.tenant_id;

ALTER TABLE finance_payments ALTER COLUMN currency SET NOT NULL;

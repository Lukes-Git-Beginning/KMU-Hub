BEGIN;

SET LOCAL row_security = off;

-- Add contact_id column to finance_invoices for Contact-360 linkage.
ALTER TABLE finance_invoices
    ADD COLUMN contact_id UUID NULL REFERENCES contacts(id) ON DELETE SET NULL;

-- Partial index for Contact-360 lookups (sparse: only rows with a contact).
CREATE INDEX idx_finance_invoices_contact ON finance_invoices(contact_id)
    WHERE contact_id IS NOT NULL;

-- Backfill: derive contact_id from the linked quote's deal.
-- Path: finance_invoices.source_quote_id → finance_quotes.deal_id → deals.contact_id
UPDATE finance_invoices fi
SET contact_id = d.contact_id
FROM finance_quotes q
JOIN deals d ON d.id = q.deal_id
WHERE fi.source_quote_id = q.id
  AND fi.contact_id IS NULL
  AND d.contact_id IS NOT NULL;

COMMIT;

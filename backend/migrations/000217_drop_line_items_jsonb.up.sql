-- Migration 000217: Drop the legacy JSONB line_items column from finance documents.
-- All data was migrated to the relational line tables in 000132/000133.
-- All application code now reads exclusively from finance_invoice_lines,
-- finance_quote_lines, and finance_credit_note_lines.
-- NOTE: finance_incoming_invoices.line_items is intentionally NOT dropped here
-- as it has no relational replacement.

ALTER TABLE finance_invoices DROP COLUMN IF EXISTS line_items;
ALTER TABLE finance_quotes DROP COLUMN IF EXISTS line_items;
ALTER TABLE finance_credit_notes DROP COLUMN IF EXISTS line_items;

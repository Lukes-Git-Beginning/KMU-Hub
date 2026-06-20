-- Restore JSONB line_items columns (nullable — data is not restored).
ALTER TABLE finance_invoices ADD COLUMN IF NOT EXISTS line_items JSONB;
ALTER TABLE finance_quotes ADD COLUMN IF NOT EXISTS line_items JSONB;
ALTER TABLE finance_credit_notes ADD COLUMN IF NOT EXISTS line_items JSONB;

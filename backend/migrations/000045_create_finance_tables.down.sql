-- Drop finance tables in reverse dependency order

DROP INDEX IF EXISTS idx_finance_dunning_invoice;
DROP INDEX IF EXISTS idx_finance_payments_invoice;
DROP INDEX IF EXISTS idx_finance_credit_notes_invoice;
DROP INDEX IF EXISTS idx_finance_invoices_number;
DROP INDEX IF EXISTS idx_finance_invoices_due_date;
DROP INDEX IF EXISTS idx_finance_invoices_tenant_status;
DROP INDEX IF EXISTS idx_finance_quotes_deal;
DROP INDEX IF EXISTS idx_finance_quotes_tenant_status;

DROP TABLE IF EXISTS finance_dunning_config;
DROP TABLE IF EXISTS finance_dunning_records;
DROP TABLE IF EXISTS finance_payments;
DROP TABLE IF EXISTS finance_credit_notes;
DROP TABLE IF EXISTS finance_invoices;
DROP TABLE IF EXISTS finance_quotes;
DROP TABLE IF EXISTS finance_number_sequences;
DROP TABLE IF EXISTS company_settings;

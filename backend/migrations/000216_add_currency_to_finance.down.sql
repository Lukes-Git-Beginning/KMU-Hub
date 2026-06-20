ALTER TABLE company_settings     DROP COLUMN IF EXISTS default_currency;
ALTER TABLE finance_quotes       DROP COLUMN IF EXISTS currency;
ALTER TABLE finance_credit_notes DROP COLUMN IF EXISTS currency;
ALTER TABLE finance_invoices     DROP COLUMN IF EXISTS currency;

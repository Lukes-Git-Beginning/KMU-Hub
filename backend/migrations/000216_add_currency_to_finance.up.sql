-- B6 (Welle 5): add a persisted currency to outbound finance documents plus a
-- tenant-level default, replacing hardcoded "EUR" literals in the ZUGFeRD/Lexware/
-- Bexio mappers with a settings-driven value. EUR is retroactively correct for the
-- existing EUR-only data. currency is set at create time and never mutated (GoBD).
-- Plain column additions — no RLS policy change required (tenant_id untouched).
ALTER TABLE finance_invoices     ADD COLUMN currency CHAR(3) NOT NULL DEFAULT 'EUR';
ALTER TABLE finance_credit_notes ADD COLUMN currency CHAR(3) NOT NULL DEFAULT 'EUR';
ALTER TABLE finance_quotes       ADD COLUMN currency CHAR(3) NOT NULL DEFAULT 'EUR';
ALTER TABLE company_settings     ADD COLUMN default_currency CHAR(3) NOT NULL DEFAULT 'EUR';

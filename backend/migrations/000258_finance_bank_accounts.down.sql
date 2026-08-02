-- Rollback of migration 000258: bank accounts (Bankkonten-Stammdaten).

BEGIN;

DROP INDEX IF EXISTS idx_finance_bank_statements_tenant_iban;
DROP TABLE IF EXISTS finance_bank_accounts;

COMMIT;

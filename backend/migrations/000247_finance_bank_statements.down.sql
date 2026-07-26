-- Rollback of migration 000247: bank statement import.

BEGIN;

DROP TABLE IF EXISTS finance_bank_transactions;
DROP TABLE IF EXISTS finance_bank_statements;

COMMIT;

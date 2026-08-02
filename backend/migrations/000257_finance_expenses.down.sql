-- Rollback of migration 000257: expenses (Ausgaben).

BEGIN;

DROP TABLE IF EXISTS finance_expenses;

COMMIT;

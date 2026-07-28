-- Rollback 000245: drop the report documents table.

BEGIN;

DROP TABLE IF EXISTS report_documents;

COMMIT;

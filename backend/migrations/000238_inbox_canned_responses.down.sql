-- Rollback 000238: drop the inbox canned-responses table.

BEGIN;

DROP TABLE IF EXISTS inbox_canned_responses;

COMMIT;

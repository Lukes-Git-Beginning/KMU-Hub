-- Rollback 000237: drop the inbox thread table.

BEGIN;

DROP TABLE IF EXISTS inbox_thread_messages;

COMMIT;

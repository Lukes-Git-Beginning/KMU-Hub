-- Reverses 000298. Dropping the table drops its policy and index with it.
BEGIN;

DROP TABLE IF EXISTS contract_events;

COMMIT;

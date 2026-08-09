-- Reverses 000303. Dropping the table drops its policy, index and unique
-- constraint with it.
BEGIN;

DROP TABLE IF EXISTS automation_time_trigger_fires;

COMMIT;

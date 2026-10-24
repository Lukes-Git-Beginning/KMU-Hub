BEGIN;

DROP INDEX IF EXISTS idx_trip_logs_driver_user_id;
ALTER TABLE trip_logs DROP COLUMN driver_user_id;

COMMIT;

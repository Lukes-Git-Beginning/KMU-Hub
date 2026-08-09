BEGIN;

ALTER TABLE trip_logs DROP COLUMN business_partner;

COMMIT;

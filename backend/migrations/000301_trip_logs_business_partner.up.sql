-- ============================================================================
-- Migration 000301: trip_logs.business_partner
--
-- A compliant Fahrtenbuch (logbook, section 6 EStG) needs the visited
-- business partner alongside date, mileage, destination, purpose and driver.
-- trip_logs has purpose (free text) and end_location (destination) but no
-- dedicated party field -- the export in this same commit needs it as its
-- own column, not folded into purpose where it would be unrecoverable for
-- existing rows. NOT NULL DEFAULT '' so the column reads as "not recorded"
-- rather than NULL on every row written before this migration.
-- ============================================================================

BEGIN;

ALTER TABLE trip_logs ADD COLUMN business_partner TEXT NOT NULL DEFAULT '';

COMMIT;

-- ============================================================================
-- Migration 000340: trip_logs.driver_user_id
--
-- A Fahrtenbuch entry named its driver only as free text (driver_name), so a
-- DSAR for an employee could not find their trips and a rename split the
-- history. driver_user_id links the entry to the account; driver_name stays
-- for external drivers and as the printed name. NULL = not linked.
--
-- The FK does not check the tenant: the fuhrpark service validates that the
-- user belongs to the caller's tenant before it writes the column. No backfill
-- by name -- two people with the same name would be linked wrongly.
-- ============================================================================

BEGIN;

ALTER TABLE trip_logs ADD COLUMN driver_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX idx_trip_logs_driver_user_id ON trip_logs (tenant_id, driver_user_id) WHERE driver_user_id IS NOT NULL;

COMMIT;

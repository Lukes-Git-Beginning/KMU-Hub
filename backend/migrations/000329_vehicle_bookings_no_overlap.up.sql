-- Migration 000329: DB-level overlap prevention for vehicle bookings.
--
-- CreateBookingWithLock serialises inserts per (tenant, vehicle) with an advisory lock,
-- but UpdateVehicleBooking checks conflicts outside any transaction and without that
-- lock, so two concurrent moves or reactivations (or a move racing an insert) could
-- double-book a vehicle. The exclusion constraint closes every write path at once; the
-- repository maps the resulting 23P01 to ErrBookingConflict.
--
-- The header of 000300 rejected EXCLUDE because "btree_gist is not guaranteed available
-- in every deployment target". That no longer holds: 000034, 000101 and 000328 already
-- create the extension, so it exists wherever this migration runs.
--
-- Fails if overlapping active bookings already exist — resolve those first
-- (see the pre-check query in the backlog unit harden-vehicle-booking-exclusion-constraint).
-- tstzrange is half-open: [start, end), matching bookingOverlapPredicate in the repository.
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE vehicle_bookings
    ADD CONSTRAINT vehicle_bookings_no_overlap
    EXCLUDE USING GIST (
        tenant_id  WITH =,
        vehicle_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    )
    WHERE (status IN ('booked', 'in_use'));

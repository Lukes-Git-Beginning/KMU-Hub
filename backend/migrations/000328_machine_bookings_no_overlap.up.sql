-- Migration 000328: DB-level overlap prevention for machine bookings.
--
-- CreateBookingWithLock serialises inserts per (tenant, machine) with an advisory lock,
-- but UpdateMachineBooking checks conflicts outside any transaction and without that
-- lock, so two concurrent moves (or a move racing an insert) could double-book a
-- machine. The exclusion constraint closes every write path at once; the repository
-- maps the resulting 23P01 to ErrBookingConflict.
--
-- Fails if overlapping active bookings already exist — resolve those first
-- (see the pre-check query in the backlog unit harden-machine-booking-exclusion-constraint).
-- tstzrange is half-open: [start, end), matching the application-side conflict check.
CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE machine_bookings
    ADD CONSTRAINT machine_bookings_no_overlap
    EXCLUDE USING GIST (
        tenant_id  WITH =,
        machine_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    )
    WHERE (status IN ('booked', 'in_use'));

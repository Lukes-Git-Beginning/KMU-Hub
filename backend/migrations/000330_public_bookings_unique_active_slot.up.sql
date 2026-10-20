-- Migration 000330: one active public booking per slot of a booking page.
--
-- public_bookings_unique_slot (000135) includes customer_email, so it only stops the
-- same customer from submitting twice. CreatePublicBooking checks the slot against
-- calendar events, but the event is created after the insert and best-effort, so two
-- concurrent customers (or a second customer after a failed event creation) could take
-- the same slot. The repository maps the resulting 23505 to ErrBookingSlotUnavailable.
--
-- staff_user_id is not part of the key: GetAvailability hands out each slot of a page
-- once, regardless of staff, and no code path assigns staff yet.
--
-- Fails if duplicate active bookings already exist — resolve those first
-- (see the pre-check query in the backlog unit harden-public-booking-slot-unique-index).
CREATE UNIQUE INDEX IF NOT EXISTS public_bookings_unique_active_slot
    ON public_bookings (booking_page_id, date, time_slot)
    WHERE status IN ('pending', 'confirmed');

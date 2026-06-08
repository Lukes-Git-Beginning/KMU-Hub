-- Rollback 000135 — drop booking pages tables
BEGIN;
DROP TABLE IF EXISTS public_bookings;
DROP TABLE IF EXISTS booking_pages;
COMMIT;

-- Migration 000149 (down): drop the global active-slug unique index.
BEGIN;
DROP INDEX idx_booking_pages_active_slug;
COMMIT;

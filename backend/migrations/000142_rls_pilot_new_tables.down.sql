-- Rollback for 000142: drop policies and disable RLS on the three tables.
-- Helper functions and procedures from 000118 stay in place.

SET LOCAL row_security = off;

BEGIN;

-- public_bookings
DROP POLICY IF EXISTS tenant_isolation ON public_bookings;
ALTER TABLE public_bookings NO FORCE ROW LEVEL SECURITY;
ALTER TABLE public_bookings DISABLE ROW LEVEL SECURITY;

-- booking_pages
DROP POLICY IF EXISTS tenant_isolation ON booking_pages;
ALTER TABLE booking_pages NO FORCE ROW LEVEL SECURITY;
ALTER TABLE booking_pages DISABLE ROW LEVEL SECURITY;

-- password_reset_tokens
DROP POLICY IF EXISTS tenant_isolation ON password_reset_tokens;
ALTER TABLE password_reset_tokens NO FORCE ROW LEVEL SECURITY;
ALTER TABLE password_reset_tokens DISABLE ROW LEVEL SECURITY;

COMMIT;

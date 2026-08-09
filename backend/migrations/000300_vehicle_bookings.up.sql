-- ============================================================================
-- Migration 000300: vehicle_bookings -- pool reservations for fleet vehicles.
--
-- The fuhrpark module tracks what happened to a vehicle (services, damages,
-- fuel, trips) but nothing about who has it next. Twenty-seven
-- /api/v1/fuhrpark/* paths, no reservation among them: a pool car is booked
-- in a spreadsheet today, which is exactly where double-bookings come from.
--
-- Shape follows machine_bookings (000087), the reservation table this repo
-- already has, down to the status vocabulary -- there is no reason for a car
-- and a machine to disagree about what "booked" means. Two deliberate
-- differences:
--   * vehicle_id is a real FK. machine_bookings.machine_id is TEXT because
--     produktion has no machines table; vehicles exists, so the database can
--     hold the reference instead of the service layer.
--   * user_id is the person the vehicle is reserved for and is required. A
--     reservation nobody is responsible for cannot be enforced at the key
--     cabinet. ON DELETE CASCADE: a deleted account's future bookings should
--     free the vehicle, not block it.
--
-- Overlap is checked in SQL as starts_at < :ends_at AND ends_at > :starts_at
-- (half-open [starts_at, ends_at)), not with an EXCLUDE constraint: btree_gist
-- is not guaranteed available in every deployment target, and the check has to
-- run inside the advisory-lock transaction anyway. Same reasoning as the
-- comment on produktion.FindConflictingBooking.
-- ============================================================================

BEGIN;

CREATE TABLE vehicle_bookings (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    vehicle_id UUID        NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    starts_at  TIMESTAMPTZ NOT NULL,
    ends_at    TIMESTAMPTZ NOT NULL,
    purpose    TEXT        NOT NULL DEFAULT '',
    status     TEXT        NOT NULL DEFAULT 'booked'
                             CHECK (status IN ('booked', 'in_use', 'completed', 'cancelled')),
    created_by UUID        NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);

-- Serves both reads: the conflict probe (tenant + vehicle + interval) and the
-- calendar list, which orders by starts_at. tenant_id leads because RLS puts
-- it into every query anyway.
CREATE INDEX idx_vehicle_bookings_vehicle_window
    ON vehicle_bookings (tenant_id, vehicle_id, starts_at);

CALL enable_tenant_rls('vehicle_bookings');

-- Guard seed for the new fuhrpark:booking resource. Without this every caller
-- -- including admin -- gets a 403 on the routes added in the same commit.
DO $$
DECLARE v_admin_role_id UUID;
BEGIN
    SELECT id INTO v_admin_role_id FROM roles WHERE name = 'admin' LIMIT 1;
    IF v_admin_role_id IS NOT NULL THEN
        INSERT INTO permissions (name, resource, action) VALUES
            ('fuhrpark:booking:read',  'fuhrpark:booking', 'read'),
            ('fuhrpark:booking:write', 'fuhrpark:booking', 'write')
        ON CONFLICT (name) DO NOTHING;
        INSERT INTO role_permissions (role_id, permission_id)
        SELECT v_admin_role_id, p.id FROM permissions p
        WHERE p.name IN ('fuhrpark:booking:read', 'fuhrpark:booking:write')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

COMMIT;

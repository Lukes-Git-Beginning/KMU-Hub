-- Seed meetings:write permission required by the POST /api/v1/meetings and
-- POST /api/v1/meetings/{id}/start|end permission guards added in S4.5/R2-P1.5.
-- Without this row the guarded routes return 403 for ALL users including admins.
--
-- meetings:write covers create, update, delete, start, and end operations.
-- Read access (GET) intentionally does not require a permission — any
-- authenticated tenant member may list/view meetings.

INSERT INTO permissions (name, resource, action) VALUES
    ('meetings:write', 'meetings', 'write')
ON CONFLICT (name) DO NOTHING;

-- Grant to admin, manager AND member: creating/organizing meetings is core
-- day-to-day collaboration (unlike the sensitive modules seeded admin-only in
-- 000129). Before this guard existed every authenticated user could create
-- meetings — seeding admin-only would silently remove that capability.
-- Start/end of foreign meetings is already protected by the organizer check
-- in the meeting service (ErrNotOrganizer).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('admin', 'manager', 'member')
  AND p.name = 'meetings:write'
ON CONFLICT (role_id, permission_id) DO NOTHING;

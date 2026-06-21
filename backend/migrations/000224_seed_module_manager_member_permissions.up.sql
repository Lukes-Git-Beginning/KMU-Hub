-- Grant manager + member roles their operational permissions for the
-- booking-pages, schichten, hr:time_*, inventar, and einkauf modules.
--
-- Every permission for these modules was seeded admin-only (migrations
-- 084/086/095/136/161/178/179/181/185/209), so until now every manager and
-- member hit 403 on all of these routes — an operational blocker, since the
-- manager is meant to run these modules day-to-day. The permissions already
-- exist; only the role_permissions rows are missing.
--
-- Product decision (2026-06-21):
--   manager → full operational access (read/write/create/approve/delete) on all
--             five module groups ("module operator"), including booking-pages
--             delete and shift management (shift/template/assignment/swap).
--   member  → self-service only: request/view shift swaps, view booking pages,
--             read-only inventory. No write, no purchasing, no shift/template
--             config, no swap approval.

-- Manager: every permission whose resource belongs to one of the five modules.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.resource IN (
    'booking-pages',
    'schichten:shift', 'schichten:template', 'schichten:assignment', 'schichten:swap',
    'hr:time_category', 'hr:time_template', 'hr:time_project',
    'inventar:item', 'inventar:movement', 'inventar:warning', 'inventar:location', 'inventar:inventur',
    'einkauf:po', 'einkauf:line', 'einkauf:supplier', 'einkauf:catalog', 'einkauf:contract', 'einkauf:rating'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Member: self-service subset only.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'member'
  AND p.name IN (
    'schichten:swap:read', 'schichten:swap:create',
    'booking-pages:read',
    'inventar:item:read', 'inventar:movement:read', 'inventar:warning:read',
    'inventar:location:read', 'inventar:inventur:read'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Self-service keys of the Fuhrpark, Produktion and Vermietung modules for member and manager.
--
-- Before this, a member could report a damage and log fuel/trips but could not book a pool vehicle,
-- read the damages they reported, file their own driver license check, read the work steps of an
-- order they may already start/complete (workstep:edit, order:read) or read a rental's inspection
-- reports (they can create one). Each of those answered 403.
--
-- Fuhrpark booking/license/damage:read are the existing keys (migrations 097, 279, 300), granted at
-- scope 'own'. The gateway turns 'own' into a filter on the caller (bookings: user_id, licenses:
-- driver_id, damages: reported_by) and the fuhrpark service refuses updates/deletes of a record
-- that belongs to someone else (NotFound). damage:write (update/resolve) and the vehicle-wide
-- lists stay with admin. Work steps and inspections are children of orders/rentals that
-- member and manager already read tenant-wide, so they are granted at 'all'.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('manager', 'member')
  AND p.name IN ('fuhrpark:booking:read', 'fuhrpark:booking:write',
                 'fuhrpark:license:read', 'fuhrpark:license:write',
                 'fuhrpark:damage:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('manager', 'member')
  AND p.name IN ('produktion:workstep:read', 'vermietung:inspection:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

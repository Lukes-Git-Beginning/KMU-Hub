-- Migration 000251: the platform role that may create tenants.
--
-- POST /api/v1/tenants is the one route in the system that writes outside the
-- caller's tenant, so it deliberately does not run on RequireRole("admin"):
-- a tenant administrator administers their own tenant, and handing them the
-- ability to create more would make every customer a reseller.
--
-- platform_admin is a role nobody holds after this migration. It is assigned
-- by hand in the database, and it cannot be handed out over the API — the
-- invitation and role-assignment handlers validate oneof=admin manager member,
-- so neither an invitation nor POST /users/{id}/roles can name it.
--
-- Without the permission rows RequirePermission("tenants", "write") answers 403
-- for everyone including admins; the seed is what makes the route reachable.

BEGIN;

INSERT INTO roles (name, description) VALUES
    ('platform_admin', 'Platform operator: may provision tenants. Assigned in the database only.')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (name, resource, action) VALUES
    ('tenants:write', 'tenants', 'write')
ON CONFLICT (name) DO NOTHING;

-- Only platform_admin. Explicitly not admin or manager: a tenant admin holding
-- tenants:write could provision tenants without limit, and the seat and plan
-- caps that make the licence model mean anything sit per tenant.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'platform_admin'
  AND p.name = 'tenants:write'
ON CONFLICT (role_id, permission_id) DO NOTHING;

COMMIT;

-- manager sees the finance module tile (decision 2026-10-25,
-- seed-finance-module-view-for-managers). manager holds finance:read and finance:write
-- (coarse keys, scope all) but never finance:module:view, so the Desktop module tile stayed
-- hidden although every finance route is open to them. One row at scope all, like the other
-- module:view keys of the preset. The key exists since 000256; admin and readonly hold it.
--
-- Side effect, intended: berichte/kpis lets a caller with finance:module:view read the
-- "finanzen" tiles (revenue), which manager can already derive from the invoices they read.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.name = 'manager' AND r.tenant_id IS NULL
  AND p.name = 'finance:module:view'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Grant the manager role finance:read + finance:write.
--
-- Migration 000129 seeded the finance:* permissions but granted them to the
-- admin role only, so every manager hit 403 on all finance routes. The product
-- decision is: manager gets read + write (create/send invoices, record payments,
-- send dunnings, view dashboards) but NOT finance:delete (delete quote/payment)
-- or finance:admin (company settings, invoice lock, GoBD export, dunning config).
-- The member role gets nothing.
--
-- Permissions already exist (000129); only the role_permissions rows are missing.

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.name IN ('finance:read', 'finance:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- member writes the deals of their team (decision 2026-10-25,
-- feat-crm-deals-write-team-scope-for-members): crm:deal:create and
-- crm:deal:edit at scope team. The gateway passes the caller as editor and the
-- CRM service writes only deals whose owner (owner_id, else created_by) is the
-- caller or one of their direct reports (hr_employee_profiles.manager_user_id);
-- any other deal answers 403. Deleting stays with manager and admin
-- (crm:deal:delete). Both keys exist since 000256; admin holds them at scope all.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'team'
FROM roles r, permissions p
WHERE r.name = 'member' AND r.tenant_id IS NULL
  AND p.name IN ('crm:deal:create', 'crm:deal:edit')
ON CONFLICT (role_id, permission_id) DO NOTHING;

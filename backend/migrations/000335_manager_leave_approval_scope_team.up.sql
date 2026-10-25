-- team:absence:approve / team:absence:read for the manager preset: scope 'team' instead of 'all'.
--
-- POST /hr/leave/requests/{id}/approve|reject now also accept team:absence:approve (migration 256
-- seeded it for manager, but no route used it). Decision Luke 2026-10-24: a manager decides leave
-- requests of their reporting line (hr_employee_profiles.manager_user_id), admin and hr_admin
-- decide for everyone. The HR service restricts the decision to direct reports for any scope other
-- than 'all'.
--
-- Takes effect after the next login: scopes are baked into the JWT, and a token minted before this
-- migration still resolves both keys to 'all'.
UPDATE role_permissions rp
SET scope = 'team'
FROM roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name = 'manager'
  AND p.name IN ('team:absence:approve', 'team:absence:read');

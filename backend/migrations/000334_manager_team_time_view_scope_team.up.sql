-- zeiterfassung:team:view for the manager preset: scope 'team' instead of 'all'.
--
-- GET /hr/time/team listed every employee of the tenant for anyone holding this key, and the
-- manager preset carried it at scope 'all' (migration 256). Decision Luke 2026-10-24: a
-- manager sees only their reporting line (hr_employee_profiles.manager_user_id), admin
-- sees everyone. The service now narrows the query for any scope other than 'all'.
--
-- Takes effect after the next login: scopes are baked into the JWT, and a token minted
-- before this migration still resolves the key to 'all'.
UPDATE role_permissions rp
SET scope = 'team'
FROM roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name = 'manager'
  AND p.name = 'zeiterfassung:team:view';

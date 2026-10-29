-- manager reads the damages of every vehicle, like member since migration 360.
--
-- Migration 339 granted fuhrpark:damage:read to manager at scope 'own', and 360 lifted only member
-- to 'all'. A manager therefore saw LESS in /fuhrpark/damages and /fuhrpark/vehicles/{id}/damages
-- than a driver, while the same damages show in /vehicles/{id}/history. Decision 2026-10-29:
-- manager reads at scope 'all'. Update and resolve stay with damage:write (admin); the reporter of
-- a new damage stays the caller because the gateway ties naming a colleague to damage:write.
--
-- Takes effect after the next login: permissions are baked into the JWT.
UPDATE role_permissions rp
SET scope = 'all'
FROM roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name = 'manager'
  AND p.name = 'fuhrpark:damage:read';

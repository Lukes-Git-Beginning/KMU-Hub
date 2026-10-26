-- member reads the damages of every vehicle, not only the ones they reported.
--
-- Migration 339 granted fuhrpark:damage:read to member at scope 'own', so the damage lists were
-- narrowed to reported_by = caller, while the same damages already showed up in
-- /vehicles/{id}/history (fuhrpark:vehicle:read). A driver should see the known damages of the
-- pool vehicle before taking it. Update and resolve stay with damage:write (admin). Filing a
-- damage under a colleague is now tied to damage:write in the gateway instead of the read scope,
-- so the reporter stays the caller. manager keeps scope 'own' (not part of the decision).
--
-- Takes effect after the next login: permissions are baked into the JWT.
UPDATE role_permissions rp
SET scope = 'all'
FROM roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.tenant_id IS NULL
  AND r.name = 'member'
  AND p.name = 'fuhrpark:damage:read';

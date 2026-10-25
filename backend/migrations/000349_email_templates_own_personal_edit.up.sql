-- Email templates: own personal ones free, shared ones admin only (decision 2026-10-25).
--
-- PUT and DELETE /email/templates/{id} were guarded by the coarse email:write/email:delete keys
-- (admin only), because the template service let anyone who could see a shared template change,
-- delete or take it over. The service now leaves shared templates and visibility changes to the
-- coarse keys and confines everyone else to their own personal templates, so the new
-- email:template:write|delete keys are safe for member and manager at scope 'own'; admin gets them
-- at 'all'. Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO permissions (name, resource, action) VALUES
    ('email:template:write', 'email:template', 'write'),
    ('email:template:delete', 'email:template', 'delete')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name IN ('member', 'manager') AND r.tenant_id IS NULL
  AND p.name IN ('email:template:write', 'email:template:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.name = 'admin' AND r.tenant_id IS NULL
  AND p.name IN ('email:template:write', 'email:template:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

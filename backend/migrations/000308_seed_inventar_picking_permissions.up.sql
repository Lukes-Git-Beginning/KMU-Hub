-- 000308 Permissions fuer die Kommissionierung (picking_lists).
-- Ohne diesen Seed antworten die neuen /inventar/picking-Routen jedem 403,
-- Admin eingeschlossen — genau der Fall, den
-- TestEveryRouteGuardHasAUsablePermission abfaengt.

INSERT INTO permissions (name, resource, action) VALUES
    ('inventar:picking:read',  'inventar:picking', 'read'),
    ('inventar:picking:write', 'inventar:picking', 'write'),
    ('inventar:picking:book',  'inventar:picking', 'book')
ON CONFLICT (name) DO NOTHING;

-- admin und manager fuehren Picklisten, member liest sie nur.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('admin', 'manager')
  AND p.resource = 'inventar:picking'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name = 'member'
  AND p.name = 'inventar:picking:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

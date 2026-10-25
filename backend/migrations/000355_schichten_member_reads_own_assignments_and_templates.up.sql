-- member sees their own roster (decision 2026-10-25, feat-schichten-own-assignments-for-members):
-- schichten:assignment:read at scope own (the gateway narrows GET /shifts/{id}/assignments to
-- the caller's own assignments) and schichten:template:read at scope all (shift templates hold
-- no personal data). Both keys exist since 000095; admin, manager and readonly hold them already.
INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name = 'member' AND r.tenant_id IS NULL
  AND p.name IN ('schichten:assignment:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.name = 'member' AND r.tenant_id IS NULL
  AND p.name IN ('schichten:template:read')
ON CONFLICT (role_id, permission_id) DO NOTHING;

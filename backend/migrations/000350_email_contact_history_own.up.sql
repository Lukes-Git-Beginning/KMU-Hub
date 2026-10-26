-- Contact mail history: members see their own mail, managers and admins all (decision 2026-10-25).
--
-- GET /email/links/contact/{contactId} was guarded by the coarse email:read key (admin only), so
-- the mail tile on a contact stayed empty for everyone else. The gateway now also accepts the new
-- email:contact_history:read key; at scope 'own' it asks the email server for messages of the
-- caller's own mailboxes only, and the filter sits in the query (page and total included).
-- member gets the key at 'own', manager and admin at 'all'. Takes effect after the next login:
-- permissions are baked into the JWT.
INSERT INTO permissions (name, resource, action) VALUES
    ('email:contact_history:read', 'email:contact_history', 'read')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name = 'member' AND r.tenant_id IS NULL
  AND p.name = 'email:contact_history:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.name IN ('manager', 'admin') AND r.tenant_id IS NULL
  AND p.name = 'email:contact_history:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

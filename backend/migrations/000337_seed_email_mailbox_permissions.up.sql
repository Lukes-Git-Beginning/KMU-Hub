-- email:mailbox:read|write|delete for member and manager at scope 'own', admin at 'all'.
--
-- Migration 129 gave email:read|write|delete to admin only, so the mail module answered 403 to
-- every other employee. The coarse keys also guard routes that reach beyond one mailbox (account
-- provisioning that dials any host, tenant-wide rules, the label catalogue, shared templates,
-- the contact mail history, CRM import/export), so they stay admin-only. The new keys open only
-- the mailbox routes, and the email server confines each of those to the caller's own accounts
-- (ownAccount/ownMessage/ownSignature in server/email_grpc.go). 'own' records that intent.
--
-- Takes effect after the next login: permissions are baked into the JWT.
INSERT INTO permissions (name, resource, action) VALUES
    ('email:mailbox:read', 'email:mailbox', 'read'),
    ('email:mailbox:write', 'email:mailbox', 'write'),
    ('email:mailbox:delete', 'email:mailbox', 'delete')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name IN ('manager', 'member')
  AND p.name IN ('email:mailbox:read', 'email:mailbox:write', 'email:mailbox:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.tenant_id IS NULL
  AND r.name = 'admin'
  AND p.name IN ('email:mailbox:read', 'email:mailbox:write', 'email:mailbox:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

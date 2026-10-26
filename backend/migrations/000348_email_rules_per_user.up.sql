-- Email rules per user (decision 2026-10-25: rules per user; existing rules stay tenant-wide admin
-- rules).
--
-- email_rules had no owner, and ApplyAll ran every rule over the messages of every mailbox of the
-- tenant. user_id NULL keeps today's meaning: a tenant-wide rule, maintained by admin only and
-- applied to every mailbox. A rule with user_id applies to that user's mailboxes only. Every rule
-- created from now on carries its creator. The user boundary sits in the rule service; RLS stays
-- tenant-based.
--
-- email:rule:read|write|delete open the rule routes for member and manager at scope 'own' (their
-- own rules plus reading the tenant-wide ones), admin at 'all'. Takes effect after the next login:
-- permissions are baked into the JWT.
ALTER TABLE email_rules
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_email_rules_tenant_user ON email_rules (tenant_id, user_id);

INSERT INTO permissions (name, resource, action) VALUES
    ('email:rule:read', 'email:rule', 'read'),
    ('email:rule:write', 'email:rule', 'write'),
    ('email:rule:delete', 'email:rule', 'delete')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'own'
FROM roles r, permissions p
WHERE r.name IN ('member', 'manager') AND r.tenant_id IS NULL
  AND p.name IN ('email:rule:read', 'email:rule:write', 'email:rule:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, scope)
SELECT r.id, p.id, 'all'
FROM roles r, permissions p
WHERE r.name = 'admin' AND r.tenant_id IS NULL
  AND p.name IN ('email:rule:read', 'email:rule:write', 'email:rule:delete')
ON CONFLICT (role_id, permission_id) DO NOTHING;

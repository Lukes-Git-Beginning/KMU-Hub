-- Remove the email:rule keys with every grant of them, and the rule owner column. Rules owned by a
-- user are deleted first: without the column they would turn into tenant-wide rules and run over
-- every mailbox.
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('email:rule:read', 'email:rule:write', 'email:rule:delete')
);

DELETE FROM permissions WHERE name IN ('email:rule:read', 'email:rule:write', 'email:rule:delete');

DELETE FROM email_rules WHERE user_id IS NOT NULL;

DROP INDEX IF EXISTS idx_email_rules_tenant_user;

ALTER TABLE email_rules DROP COLUMN IF EXISTS user_id;

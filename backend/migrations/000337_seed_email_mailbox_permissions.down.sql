-- Remove the email:mailbox keys and every grant of them added by the up migration.
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE name IN ('email:mailbox:read', 'email:mailbox:write', 'email:mailbox:delete')
);

DELETE FROM permissions
WHERE name IN ('email:mailbox:read', 'email:mailbox:write', 'email:mailbox:delete');

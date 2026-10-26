-- Remove the email:account keys and every grant of them added by the up migration.
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('email:account:write', 'email:account:delete')
);

DELETE FROM permissions WHERE name IN ('email:account:write', 'email:account:delete');

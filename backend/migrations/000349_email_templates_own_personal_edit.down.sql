-- Remove the email:template keys with every grant of them.
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('email:template:write', 'email:template:delete')
);

DELETE FROM permissions WHERE name IN ('email:template:write', 'email:template:delete');

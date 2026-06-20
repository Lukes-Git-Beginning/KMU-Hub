DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('module-grants:read', 'module-grants:write')
);

DELETE FROM permissions WHERE name IN ('module-grants:read', 'module-grants:write');

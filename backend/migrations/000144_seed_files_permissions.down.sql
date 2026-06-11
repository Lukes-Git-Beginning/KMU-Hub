-- Rollback for 000144: remove files:* role grants and permissions.

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('files:read', 'files:write')
);

DELETE FROM permissions WHERE name IN ('files:read', 'files:write');

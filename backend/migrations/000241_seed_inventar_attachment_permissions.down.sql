DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('inventar:attachment:read', 'inventar:attachment:write')
);
DELETE FROM permissions WHERE name IN ('inventar:attachment:read', 'inventar:attachment:write');

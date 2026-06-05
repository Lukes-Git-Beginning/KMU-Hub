DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE name = 'meetings:write');

DELETE FROM permissions WHERE name = 'meetings:write';

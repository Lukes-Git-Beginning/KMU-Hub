DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'inventar:picking'
);
DELETE FROM permissions WHERE resource = 'inventar:picking';

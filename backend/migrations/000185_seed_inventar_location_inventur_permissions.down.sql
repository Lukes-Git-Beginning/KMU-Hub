DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource IN ('inventar:location', 'inventar:inventur')
);
DELETE FROM permissions WHERE resource IN ('inventar:location', 'inventar:inventur');

-- 000209 Permissions für Einkauf Catalog, Ratings und Framework Contracts — revert

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE resource IN ('einkauf:catalog', 'einkauf:rating', 'einkauf:contract')
);

DELETE FROM permissions
WHERE name IN (
    'einkauf:catalog:read',
    'einkauf:catalog:write',
    'einkauf:rating:read',
    'einkauf:rating:write',
    'einkauf:contract:read',
    'einkauf:contract:write'
);

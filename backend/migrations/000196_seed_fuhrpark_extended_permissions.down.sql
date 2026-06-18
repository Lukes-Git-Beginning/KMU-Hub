DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN (
        'fuhrpark:fuel:read','fuhrpark:fuel:write',
        'fuhrpark:trip:read','fuhrpark:trip:write',
        'fuhrpark:document:read','fuhrpark:document:write',
        'fuhrpark:gps:read','fuhrpark:gps:write'
    )
);
DELETE FROM permissions WHERE name IN (
    'fuhrpark:fuel:read','fuhrpark:fuel:write',
    'fuhrpark:trip:read','fuhrpark:trip:write',
    'fuhrpark:document:read','fuhrpark:document:write',
    'fuhrpark:gps:read','fuhrpark:gps:write'
);

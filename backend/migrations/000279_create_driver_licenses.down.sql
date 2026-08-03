DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('fuhrpark:license:read', 'fuhrpark:license:write')
);
DELETE FROM permissions WHERE name IN ('fuhrpark:license:read', 'fuhrpark:license:write');

DROP TABLE IF EXISTS driver_licenses;

-- Reverses 000300. Dropping the table drops its policy and index with it.
BEGIN;

DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('fuhrpark:booking:read', 'fuhrpark:booking:write')
);
DELETE FROM permissions WHERE name IN ('fuhrpark:booking:read', 'fuhrpark:booking:write');

DROP TABLE IF EXISTS vehicle_bookings;

COMMIT;

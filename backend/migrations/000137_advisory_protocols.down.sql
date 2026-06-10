-- Migration 000137 — Advisory Protocols (rollback)

-- Remove permission grants first
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE name IN ('advisory-protocols:read', 'advisory-protocols:write', 'advisory-protocols:delete')
);

DELETE FROM permissions
WHERE name IN ('advisory-protocols:read', 'advisory-protocols:write', 'advisory-protocols:delete');

-- Drop table and type
DROP TABLE IF EXISTS advisory_protocols;
DROP TYPE IF EXISTS advisory_status;

-- Revert contacts extension columns
ALTER TABLE contacts
    DROP COLUMN IF EXISTS referred_by_contact_id,
    DROP COLUMN IF EXISTS client_segment;

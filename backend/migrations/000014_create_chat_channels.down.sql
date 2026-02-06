-- Remove channels permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'channels'
);
DELETE FROM permissions WHERE resource = 'channels';

-- Drop tables
DROP TABLE IF EXISTS channel_memberships;
DROP TABLE IF EXISTS channels;

-- Drop enum
DROP TYPE IF EXISTS channel_role;

-- Remove messages permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'messages'
);
DELETE FROM permissions WHERE resource = 'messages';

-- Drop tables
DROP TABLE IF EXISTS messages;

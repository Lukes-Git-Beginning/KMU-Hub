-- Remove role permissions for files
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'files'
);

-- Remove file permissions
DELETE FROM permissions WHERE resource = 'files';

-- Drop tables
DROP TABLE IF EXISTS storage_quotas;
DROP TABLE IF EXISTS chat_files;

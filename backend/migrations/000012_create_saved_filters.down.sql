-- Remove saved_filters and reports permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource IN ('saved_filters', 'reports')
);
DELETE FROM permissions WHERE resource IN ('saved_filters', 'reports');

-- Drop table
DROP TABLE IF EXISTS saved_filters;

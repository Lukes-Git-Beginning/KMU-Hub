-- Rollback projects foundation

-- Remove role permissions for projects and tasks
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource IN ('projects', 'tasks')
);

-- Remove permissions
DELETE FROM permissions WHERE resource IN ('projects', 'tasks');

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS project_statuses;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;

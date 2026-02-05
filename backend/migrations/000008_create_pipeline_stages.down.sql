-- Rollback pipeline stages

-- Remove role permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'pipeline_stages'
);

-- Remove permissions
DELETE FROM permissions WHERE resource = 'pipeline_stages';

-- Drop the table (this will cascade to deals if they exist)
DROP TABLE IF EXISTS pipeline_stages;

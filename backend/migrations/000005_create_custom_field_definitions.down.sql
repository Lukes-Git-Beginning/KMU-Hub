-- Remove custom_fields permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'custom_fields'
);
DELETE FROM permissions WHERE resource = 'custom_fields';

-- Drop custom field definitions table
DROP TABLE IF EXISTS custom_field_definitions;

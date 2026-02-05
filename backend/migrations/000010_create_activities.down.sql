-- Remove activities permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'activities'
);
DELETE FROM permissions WHERE resource = 'activities';

-- Drop FK constraint from activity_tags
ALTER TABLE activity_tags DROP CONSTRAINT IF EXISTS fk_activity_tags_activity;

-- Drop tables
DROP TABLE IF EXISTS activity_custom_field_values;
DROP TABLE IF EXISTS activities;

-- Drop enum
DROP TYPE IF EXISTS activity_type;

-- Remove tags permissions from roles
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'tags'
);

-- Remove tags permissions
DELETE FROM permissions WHERE resource = 'tags';

-- Drop junction tables
DROP TABLE IF EXISTS activity_tags;
DROP TABLE IF EXISTS deal_tags;
DROP TABLE IF EXISTS company_tags;
DROP TABLE IF EXISTS contact_tags;

-- Drop tags table
DROP TABLE IF EXISTS tags;

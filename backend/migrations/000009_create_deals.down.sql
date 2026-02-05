-- Rollback deals

-- Remove FK constraint from deal_tags
ALTER TABLE deal_tags
    DROP CONSTRAINT IF EXISTS fk_deal_tags_deal;

-- Remove role permissions
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'deals'
);

-- Remove permissions
DELETE FROM permissions WHERE resource = 'deals';

-- Drop custom field values table
DROP TABLE IF EXISTS deal_custom_field_values;

-- Drop deals table
DROP TABLE IF EXISTS deals;

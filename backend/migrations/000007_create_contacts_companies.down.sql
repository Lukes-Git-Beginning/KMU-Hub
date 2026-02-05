-- Remove companies permissions from roles
DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE resource = 'companies'
);

-- Remove companies permissions
DELETE FROM permissions WHERE resource = 'companies';

-- Remove FK constraints from junction tables
ALTER TABLE contact_tags DROP CONSTRAINT IF EXISTS fk_contact_tags_contact;
ALTER TABLE company_tags DROP CONSTRAINT IF EXISTS fk_company_tags_company;

-- Drop custom field values tables
DROP TABLE IF EXISTS company_custom_field_values;
DROP TABLE IF EXISTS contact_custom_field_values;

-- Drop contacts (references companies, so drop first)
DROP TABLE IF EXISTS contacts;

-- Drop companies
DROP TABLE IF EXISTS companies;

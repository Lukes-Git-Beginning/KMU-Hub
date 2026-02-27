DROP INDEX IF EXISTS idx_companies_name_trgm;
DROP INDEX IF EXISTS idx_contacts_name_trgm;

ALTER TABLE companies DROP COLUMN IF EXISTS merged_into_id;
ALTER TABLE contacts DROP COLUMN IF EXISTS merged_into_id;

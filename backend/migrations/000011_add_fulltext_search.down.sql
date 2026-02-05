-- Remove full-text search from all CRM entities

-- Activities
DROP TRIGGER IF EXISTS activities_search_update ON activities;
DROP FUNCTION IF EXISTS activities_search_vector_update();
ALTER TABLE activities DROP COLUMN IF EXISTS search_vector;

-- Deals
DROP TRIGGER IF EXISTS deals_search_update ON deals;
DROP FUNCTION IF EXISTS deals_search_vector_update();
ALTER TABLE deals DROP COLUMN IF EXISTS search_vector;

-- Companies
DROP TRIGGER IF EXISTS companies_search_update ON companies;
DROP FUNCTION IF EXISTS companies_search_vector_update();
ALTER TABLE companies DROP COLUMN IF EXISTS search_vector;

-- Contacts
DROP TRIGGER IF EXISTS contacts_search_update ON contacts;
DROP FUNCTION IF EXISTS contacts_search_vector_update();
ALTER TABLE contacts DROP COLUMN IF EXISTS search_vector;

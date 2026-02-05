-- Full-Text Search using PostgreSQL tsvector
-- Add search vectors to CRM entities with German language configuration

-- ============================================================================
-- Contacts: search on first_name, last_name, email, phone, position, notes
-- ============================================================================
ALTER TABLE contacts ADD COLUMN search_vector TSVECTOR;

CREATE INDEX idx_contacts_search ON contacts USING GIN (search_vector);

CREATE OR REPLACE FUNCTION contacts_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('german', COALESCE(NEW.first_name, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.last_name, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.email, '')), 'B') ||
        setweight(to_tsvector('german', COALESCE(NEW.phone, '')), 'C') ||
        setweight(to_tsvector('german', COALESCE(NEW.position, '')), 'C') ||
        setweight(to_tsvector('german', COALESCE(NEW.notes, '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER contacts_search_update
    BEFORE INSERT OR UPDATE ON contacts
    FOR EACH ROW EXECUTE FUNCTION contacts_search_vector_update();

-- Update existing contacts
UPDATE contacts SET updated_at = updated_at;

-- ============================================================================
-- Companies: search on name, domain, industry, city, notes
-- ============================================================================
ALTER TABLE companies ADD COLUMN search_vector TSVECTOR;

CREATE INDEX idx_companies_search ON companies USING GIN (search_vector);

CREATE OR REPLACE FUNCTION companies_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('german', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.domain, '')), 'B') ||
        setweight(to_tsvector('german', COALESCE(NEW.industry, '')), 'C') ||
        setweight(to_tsvector('german', COALESCE(NEW.city, '')), 'C') ||
        setweight(to_tsvector('german', COALESCE(NEW.notes, '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER companies_search_update
    BEFORE INSERT OR UPDATE ON companies
    FOR EACH ROW EXECUTE FUNCTION companies_search_vector_update();

-- Update existing companies
UPDATE companies SET updated_at = updated_at;

-- ============================================================================
-- Deals: search on name, notes
-- ============================================================================
ALTER TABLE deals ADD COLUMN search_vector TSVECTOR;

CREATE INDEX idx_deals_search ON deals USING GIN (search_vector);

CREATE OR REPLACE FUNCTION deals_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('german', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.notes, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER deals_search_update
    BEFORE INSERT OR UPDATE ON deals
    FOR EACH ROW EXECUTE FUNCTION deals_search_vector_update();

-- Update existing deals
UPDATE deals SET updated_at = updated_at;

-- ============================================================================
-- Activities: search on subject, description
-- ============================================================================
ALTER TABLE activities ADD COLUMN search_vector TSVECTOR;

CREATE INDEX idx_activities_search ON activities USING GIN (search_vector);

CREATE OR REPLACE FUNCTION activities_search_vector_update() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('german', COALESCE(NEW.subject, '')), 'A') ||
        setweight(to_tsvector('german', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER activities_search_update
    BEFORE INSERT OR UPDATE ON activities
    FOR EACH ROW EXECUTE FUNCTION activities_search_vector_update();

-- Enable trigram extension for fuzzy name matching
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add soft-merge tracking columns
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS merged_into_id UUID REFERENCES contacts(id);
ALTER TABLE companies ADD COLUMN IF NOT EXISTS merged_into_id UUID REFERENCES companies(id);

-- Trigram indexes for fuzzy duplicate detection
CREATE INDEX idx_contacts_name_trgm ON contacts USING gin (
    (lower(first_name) || ' ' || lower(last_name)) gin_trgm_ops
);
CREATE INDEX idx_companies_name_trgm ON companies USING gin (lower(name) gin_trgm_ops);

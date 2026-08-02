DROP INDEX IF EXISTS idx_contacts_lifecycle_stage;

ALTER TABLE contacts
    DROP CONSTRAINT IF EXISTS chk_contacts_lifecycle_stage,
    DROP CONSTRAINT IF EXISTS chk_contacts_lead_source,
    DROP CONSTRAINT IF EXISTS chk_contacts_lead_score,
    DROP CONSTRAINT IF EXISTS chk_contacts_lead_temperature,
    DROP CONSTRAINT IF EXISTS chk_contacts_lead_status;

ALTER TABLE contacts
    DROP COLUMN IF EXISTS lifecycle_stage,
    DROP COLUMN IF EXISTS lead_source,
    DROP COLUMN IF EXISTS lead_score,
    DROP COLUMN IF EXISTS lead_temperature,
    DROP COLUMN IF EXISTS lead_status,
    DROP COLUMN IF EXISTS lead_company;

DROP INDEX IF EXISTS idx_tickets_org_id;
DROP INDEX IF EXISTS idx_tickets_contact_id;

ALTER TABLE tickets
    DROP COLUMN IF EXISTS org_id,
    DROP COLUMN IF EXISTS contact_id;

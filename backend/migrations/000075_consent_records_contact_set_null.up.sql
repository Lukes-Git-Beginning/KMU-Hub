-- R1-P0.1: consent_records.contact_id FK: change delete behaviour to SET NULL
-- When a contact is deleted, consent records are preserved (for audit trail)
-- with contact_id set to NULL instead of deleting the row.
-- contact_id must be nullable for SET NULL to work.

ALTER TABLE consent_records
    DROP CONSTRAINT consent_records_contact_id_fkey;

ALTER TABLE consent_records
    ALTER COLUMN contact_id DROP NOT NULL;

ALTER TABLE consent_records
    ADD CONSTRAINT consent_records_contact_id_fkey
    FOREIGN KEY (contact_id)
    REFERENCES contacts(id)
    ON DELETE SET NULL;

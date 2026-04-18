-- Revert: SET NULL → CASCADE, restore NOT NULL
-- Note: rows with contact_id IS NULL (from deleted contacts) are removed
-- to satisfy the NOT NULL constraint being restored.

DELETE FROM consent_records WHERE contact_id IS NULL;

ALTER TABLE consent_records
    DROP CONSTRAINT consent_records_contact_id_fkey;

ALTER TABLE consent_records
    ALTER COLUMN contact_id SET NOT NULL;

ALTER TABLE consent_records
    ADD CONSTRAINT consent_records_contact_id_fkey
    FOREIGN KEY (contact_id)
    REFERENCES contacts(id)
    ON DELETE CASCADE;

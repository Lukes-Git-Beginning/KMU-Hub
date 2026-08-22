ALTER TABLE contacts DROP CONSTRAINT contacts_merged_into_id_fkey;
ALTER TABLE contacts
    ADD CONSTRAINT contacts_merged_into_id_fkey
    FOREIGN KEY (merged_into_id) REFERENCES contacts(id);

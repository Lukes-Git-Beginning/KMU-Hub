ALTER TABLE companies DROP CONSTRAINT companies_merged_into_id_fkey;
ALTER TABLE companies
    ADD CONSTRAINT companies_merged_into_id_fkey
    FOREIGN KEY (merged_into_id) REFERENCES companies(id);

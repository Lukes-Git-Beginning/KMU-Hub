-- companies.merged_into_id was created (000059) as `REFERENCES companies(id)` with no ON DELETE
-- clause, which defaults to NO ACTION. Deleting a company that serves as the primary of a
-- completed merge (another companies row has merged_into_id = its id) therefore fails the DELETE
-- with a raw FK violation instead of the clean 409 that HasContacts already gives when the
-- company still has contacts attached.
--
-- Fix is SET NULL, not a service-level guard: no read path resolves merged_into_id to redirect a
-- soft-deleted duplicate back to its primary (grep confirms merged_into_id is only used to filter
-- already-merged rows out of duplicate-candidate search, and to mark the duplicate as merged).
-- Losing the merge pointer on the duplicate row when its primary is later hard-deleted is
-- acceptable — the duplicate itself keeps existing as its own company — and this mirrors the
-- identical fix already applied to contacts.merged_into_id (000318).
ALTER TABLE companies DROP CONSTRAINT companies_merged_into_id_fkey;
ALTER TABLE companies
    ADD CONSTRAINT companies_merged_into_id_fkey
    FOREIGN KEY (merged_into_id) REFERENCES companies(id) ON DELETE SET NULL;

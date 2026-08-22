-- contacts.merged_into_id was created (000059) as `REFERENCES contacts(id)` with no ON DELETE
-- clause, which defaults to NO ACTION. Deleting a contact that serves as the primary of a
-- completed merge (another contacts row has merged_into_id = its id) therefore fails the DELETE
-- with a raw FK violation instead of the clean 409 the other RESTRICT cases get from IsInUse —
-- IsInUse never checked this FK because it wasn't in the original cascade audit.
--
-- Fix is SET NULL, not an IsInUse guard: no read path resolves merged_into_id to redirect a
-- soft-deleted duplicate back to its primary (grep confirms merged_into_id is only used to
-- filter already-merged rows out of duplicate-candidate search, and to mark the duplicate as
-- merged). Losing the merge pointer on the duplicate row when its primary is later hard-deleted
-- is acceptable — the duplicate itself keeps existing as its own contact — and SET NULL is
-- consistent with the sibling self-reference contacts.referred_by_contact_id (000137), which
-- already uses ON DELETE SET NULL for the same reason.
ALTER TABLE contacts DROP CONSTRAINT contacts_merged_into_id_fkey;
ALTER TABLE contacts
    ADD CONSTRAINT contacts_merged_into_id_fkey
    FOREIGN KEY (merged_into_id) REFERENCES contacts(id) ON DELETE SET NULL;

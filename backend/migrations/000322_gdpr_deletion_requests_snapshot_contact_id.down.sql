DROP INDEX IF EXISTS idx_gdpr_deletion_requests_original_contact_id;

ALTER TABLE gdpr_deletion_requests
    DROP CONSTRAINT IF EXISTS gdpr_deletion_requests_contact_id_fkey,
    ADD CONSTRAINT gdpr_deletion_requests_contact_id_fkey
        FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE;

-- Backfill contact_id for rows whose contact still exists (contact_id was
-- SET NULL by the up-migration's FK, not by an actual contact deletion).
-- Rows whose contact was genuinely hard-deleted after this migration was
-- applied have no surviving contacts row to reference -- original_contact_id
-- cannot be restored into contact_id for those without violating the
-- re-added FK, so they are left as-is (the audit trail this migration adds
-- is exactly the reason that data loss on down is acceptable: it is a
-- rollback of the fix, not of the underlying deletions it recorded).
UPDATE gdpr_deletion_requests SET contact_id = original_contact_id
WHERE contact_id IS NULL
  AND EXISTS (SELECT 1 FROM contacts c WHERE c.id = gdpr_deletion_requests.original_contact_id);

-- Only restore NOT NULL if every row can satisfy it; otherwise leave the
-- column nullable rather than fail the rollback outright.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM gdpr_deletion_requests WHERE contact_id IS NULL) THEN
        ALTER TABLE gdpr_deletion_requests ALTER COLUMN contact_id SET NOT NULL;
    END IF;
END $$;

ALTER TABLE gdpr_deletion_requests
    DROP COLUMN original_contact_id;

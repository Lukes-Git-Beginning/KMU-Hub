-- Revert R2-P0.9: gdpr_deletion_requests.contact_id CASCADE → no ON DELETE (RESTRICT default)
--
-- Note: rows with contact_id pointing to a non-existent contact may exist if
-- any cascade already fired. Those rows cannot be safely reverted — this down
-- migration is provided for local dev rollback only, not production use.
ALTER TABLE gdpr_deletion_requests
    DROP CONSTRAINT IF EXISTS gdpr_deletion_requests_contact_id_fkey,
    ADD CONSTRAINT gdpr_deletion_requests_contact_id_fkey
        FOREIGN KEY (contact_id) REFERENCES contacts(id);

-- Revert R2-P0.8: consent_records.created_by SET NULL → no ON DELETE (RESTRICT default)
--
-- Note: rows where created_by was set to NULL by the cascade remain NULL.
-- This down migration restores the FK behaviour only; it does not backfill
-- NULL values (the original user records are gone).
ALTER TABLE consent_records
    DROP CONSTRAINT IF EXISTS consent_records_created_by_fkey,
    ADD CONSTRAINT consent_records_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id);

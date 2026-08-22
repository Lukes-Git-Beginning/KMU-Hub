-- fix-gdpr-deletion-request-audit-trail-cascade-loss
--
-- Problem: gdpr_deletion_requests.contact_id is ON DELETE CASCADE (migration
-- 000082, "R2-P0.9"). consent.Service.ProcessDeletion never hard-deletes the
-- contact -- it only anonymizes it (AnonymizeContact), so the request row
-- normally survives. But a contact with a "completed" deletion request can
-- later be hard-deleted through a second, independent path that does not
-- know about gdpr_deletion_requests at all: contact.Service.Delete (manual)
-- or ContactRetentionHandler.Apply with action=delete (automated, the more
-- likely trigger). When that happens, the CASCADE removes the completed
-- request row along with the contact -- the one record that proves an
-- Art. 17 erasure was carried out disappears together with the data it
-- documents. That is an Art. 5(2) accountability loss, not a UI dead end.
--
-- 000082 explicitly rejected SET NULL for this column: "without the contact
-- reference it cannot be linked back to the subject and is therefore not
-- useful". That objection is addressed here, not ignored: this migration
-- adds original_contact_id, a plain UUID column with NO foreign key, that
-- is populated once at INSERT time and is therefore immune to any future
-- CASCADE/SET NULL on contacts. contact_id keeps its live FK (now SET NULL
-- instead of CASCADE) for as long as the contact row exists; once the
-- contact is gone, original_contact_id is the permanent record of which
-- contact the fulfilled request was about. That is the "dedicated snapshot"
-- 000082 already anticipated ("a dedicated gdpr_audit_log table with a
-- point-in-time snapshot should be introduced") -- implemented here as a
-- column on the existing table rather than a new one, because the existing
-- table already carries every other field (status, reason, timestamps) that
-- snapshot would need to duplicate.
ALTER TABLE gdpr_deletion_requests
    ADD COLUMN original_contact_id UUID;

UPDATE gdpr_deletion_requests SET original_contact_id = contact_id;

ALTER TABLE gdpr_deletion_requests
    ALTER COLUMN original_contact_id SET NOT NULL;

ALTER TABLE gdpr_deletion_requests
    ALTER COLUMN contact_id DROP NOT NULL;

ALTER TABLE gdpr_deletion_requests
    DROP CONSTRAINT IF EXISTS gdpr_deletion_requests_contact_id_fkey,
    ADD CONSTRAINT gdpr_deletion_requests_contact_id_fkey
        FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_gdpr_deletion_requests_original_contact_id
    ON gdpr_deletion_requests (original_contact_id);

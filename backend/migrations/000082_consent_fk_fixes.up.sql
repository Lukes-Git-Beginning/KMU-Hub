-- R2-P0.8: consent_records.created_by ON DELETE SET NULL
--
-- Reason: Consent records document historical user consent. When the
-- creating user is deleted, the consent evidence must remain as audit
-- trail with created_by set to NULL. Without this fix, user-DELETE either
-- blocks (RESTRICT default) or leaves orphan rows depending on PG version.
--
-- created_by is already nullable (no NOT NULL constraint in 000060), so
-- no column alteration is required — only the FK behaviour changes.
--
-- This is the second blind spot on consent_records; R1-P0.1 fixed
-- contact_id (SET NULL) in migration 000075.
ALTER TABLE consent_records
    DROP CONSTRAINT IF EXISTS consent_records_created_by_fkey,
    ADD CONSTRAINT consent_records_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;

-- R2-P0.9: gdpr_deletion_requests.contact_id ON DELETE CASCADE
--
-- Reason: GDPR deletion requests reference the contact targeted for
-- deletion. Without ON DELETE, a contact-DELETE is blocked by the very
-- request whose purpose is to delete that contact — a circular hard block
-- and a compliance catastrophe: the system refuses to execute a legally
-- mandated erasure.
--
-- Choice: CASCADE over SET NULL.
--   CASCADE:  when the contact is actually deleted the request has fulfilled
--             its purpose; keeping a dangling row (contact_id=NULL) provides
--             no actionable information and clutters the audit surface.
--   SET NULL: preserves the row as a "headless" audit entry, but without
--             the contact reference it cannot be linked back to the subject
--             and is therefore not useful.
-- If full audit-trail retention becomes a regulatory requirement in Sprint 3
-- (e.g. for GDPR Art. 17(3) exceptions), a dedicated gdpr_audit_log table
-- with a point-in-time snapshot should be introduced rather than weakening
-- this CASCADE to SET NULL.
ALTER TABLE gdpr_deletion_requests
    DROP CONSTRAINT IF EXISTS gdpr_deletion_requests_contact_id_fkey,
    ADD CONSTRAINT gdpr_deletion_requests_contact_id_fkey
        FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE;

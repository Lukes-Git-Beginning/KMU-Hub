-- Migration 000227: Link meetings to CRM contacts and deals (Wave 7B).
--
-- Adds optional foreign keys so a meeting can be associated with a CRM
-- contact (e.g. a sales call) and/or a deal (e.g. a negotiation session).
-- Both columns are nullable — existing meetings are unaffected.
-- meetings already has RLS enabled; column-only ALTER requires no new policy.

ALTER TABLE meetings
    ADD COLUMN contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL;

ALTER TABLE meetings
    ADD COLUMN deal_id UUID REFERENCES deals(id) ON DELETE SET NULL;

CREATE INDEX meetings_contact_id_idx ON meetings (contact_id) WHERE contact_id IS NOT NULL;
CREATE INDEX meetings_deal_id_idx    ON meetings (deal_id)    WHERE deal_id    IS NOT NULL;

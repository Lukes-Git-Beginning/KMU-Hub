-- ============================================================================
-- External requesters without a user account (BACKLOG B5 "intake-external-requester").
--
-- requester_id has been NOT NULL since 000077, which forces every requester
-- onto a tenant user. An external intake ticket -- the public form submission
-- the intake contract is being built for -- has no user row to point at, so it
-- simply cannot be stored today.
--
-- The column becomes nullable, and a CHECK keeps the two identities apart so
-- the half-empty state ("external, but nobody knows who") cannot be written:
--
--   internal -> requester_id IS NOT NULL (the user row carries name and mail)
--   external -> requester_is_external AND requester_email is present
--               (requester_email is the only way back to the person, so it is
--                the one field that is not optional for them)
--
-- Both at once stays legal on purpose: a ticket opened by an agent on behalf
-- of an external contact has a requester_id AND a reply address.
-- ============================================================================

ALTER TABLE tickets
    ALTER COLUMN requester_id DROP NOT NULL;

-- Existing rows all carry a requester_id, so they satisfy the first branch and
-- need no backfill.
ALTER TABLE tickets
    ADD CONSTRAINT chk_tickets_requester_identity CHECK (
        requester_id IS NOT NULL
        OR (requester_is_external AND requester_email IS NOT NULL AND requester_email <> '')
    );

-- Reading external tickets by reply address (dispatch, dedup) walks this.
CREATE INDEX IF NOT EXISTS idx_tickets_requester_email
    ON tickets (tenant_id, requester_email)
    WHERE requester_email IS NOT NULL;

-- tickets already carries tenant_id NOT NULL and a tenant_isolation policy;
-- loosening a column on it needs no RLS change.

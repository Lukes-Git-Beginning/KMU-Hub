-- Reverting means external requesters can no longer be represented. Rows that
-- only satisfy the external branch of the CHECK have no requester_id and would
-- block SET NOT NULL, so they are deleted first -- there is no user to migrate
-- them onto, and leaving them would make the down migration unrunnable.

DROP INDEX IF EXISTS idx_tickets_requester_email;

ALTER TABLE tickets
    DROP CONSTRAINT IF EXISTS chk_tickets_requester_identity;

DELETE FROM tickets WHERE requester_id IS NULL;

ALTER TABLE tickets
    ALTER COLUMN requester_id SET NOT NULL;

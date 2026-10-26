-- A recording now starts as 'pending' and only turns 'active' once the initiator
-- has confirmed the pre-recording dialog; the LiveKit egress is started at that
-- point, not before. room_name keeps the room captured at start so the confirm
-- step can start the egress without re-resolving the call or meeting.
ALTER TABLE recordings DROP CONSTRAINT recordings_status_check;
ALTER TABLE recordings ADD CONSTRAINT recordings_status_check
    CHECK (status IN ('pending', 'active', 'processing', 'completed', 'failed', 'deleted'));

ALTER TABLE recordings ADD COLUMN IF NOT EXISTS room_name TEXT NULL;

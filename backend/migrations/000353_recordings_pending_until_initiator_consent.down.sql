-- Pending rows never started an egress and hold no media; the old constraint
-- does not know the status, so they end as 'failed'.
UPDATE recordings SET status = 'failed' WHERE status = 'pending';

ALTER TABLE recordings DROP COLUMN IF EXISTS room_name;

ALTER TABLE recordings DROP CONSTRAINT recordings_status_check;
ALTER TABLE recordings ADD CONSTRAINT recordings_status_check
    CHECK (status IN ('active', 'processing', 'completed', 'failed', 'deleted'));

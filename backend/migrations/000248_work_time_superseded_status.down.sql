-- Superseded originals go back to 'completed' before the constraint narrows again,
-- otherwise the ADD CONSTRAINT would fail on exactly the rows this migration wrote.
UPDATE hr_work_time_entries
SET status = 'completed'
WHERE status = 'superseded';

ALTER TABLE hr_work_time_entries
    DROP CONSTRAINT IF EXISTS chk_work_time_status;

ALTER TABLE hr_work_time_entries
    ADD CONSTRAINT chk_work_time_status
    CHECK (status IN ('active', 'completed', 'correction_pending', 'correction_approved'));

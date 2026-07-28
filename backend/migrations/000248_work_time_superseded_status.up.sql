-- Approving a time correction left the original entry at 'completed', and every
-- balance query sums 'active', 'completed' and 'correction_approved' alike — so a
-- corrected day counted the original duration plus the corrected one. The original
-- is not "completed" once a correction replaced it; it is superseded, and giving
-- that state its own value takes it out of every existing sum at once instead of
-- adding an anti-join to each aggregate.

ALTER TABLE hr_work_time_entries
    DROP CONSTRAINT IF EXISTS chk_work_time_status;

ALTER TABLE hr_work_time_entries
    ADD CONSTRAINT chk_work_time_status
    CHECK (status IN ('active', 'completed', 'correction_pending', 'correction_approved', 'superseded'));

-- Backfill: every entry that an already-approved correction replaced has been
-- double-counted since the correction was approved. The join carries tenant_id so a
-- correction can only ever supersede an original of its own tenant.
UPDATE hr_work_time_entries AS original
SET status = 'superseded',
    updated_at = NOW()
FROM hr_work_time_entries AS correction
WHERE correction.original_entry_id = original.id
  AND correction.tenant_id = original.tenant_id
  AND correction.status = 'correction_approved'
  AND original.status IN ('active', 'completed');

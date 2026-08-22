DROP INDEX IF EXISTS idx_hr_work_time_entries_unbilled;

ALTER TABLE hr_work_time_entries
    DROP COLUMN IF EXISTS invoice_id,
    DROP COLUMN IF EXISTS billed_at;

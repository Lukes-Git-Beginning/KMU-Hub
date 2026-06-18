DROP INDEX IF EXISTS idx_hr_work_time_entries_project_id;
ALTER TABLE hr_work_time_entries DROP COLUMN IF EXISTS project_id;

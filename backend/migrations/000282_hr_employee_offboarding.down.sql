DROP INDEX IF EXISTS idx_hr_employee_profiles_status;

ALTER TABLE hr_employee_profiles
    DROP CONSTRAINT IF EXISTS chk_hr_employee_exit_complete,
    DROP CONSTRAINT IF EXISTS chk_hr_employee_exit_type,
    DROP CONSTRAINT IF EXISTS chk_hr_employee_status;

ALTER TABLE hr_employee_profiles
    DROP COLUMN IF EXISTS exit_reason,
    DROP COLUMN IF EXISTS exit_type,
    DROP COLUMN IF EXISTS exit_date,
    DROP COLUMN IF EXISTS last_work_day,
    DROP COLUMN IF EXISTS status;

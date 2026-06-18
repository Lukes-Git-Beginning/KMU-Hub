ALTER TABLE hr_company_settings
    DROP COLUMN IF EXISTS work_hours_per_day,
    DROP COLUMN IF EXISTS max_daily_hours,
    DROP COLUMN IF EXISTS break_after_hours;

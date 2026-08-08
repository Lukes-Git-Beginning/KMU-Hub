DROP INDEX IF EXISTS idx_hr_employee_profiles_tenant_minor;

ALTER TABLE hr_employee_profiles
    DROP COLUMN IF EXISTS is_minor;

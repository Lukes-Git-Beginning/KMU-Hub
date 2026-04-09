DROP INDEX IF EXISTS idx_hr_employee_profiles_tenant;
ALTER TABLE hr_employee_profiles DROP COLUMN IF EXISTS tenant_id;

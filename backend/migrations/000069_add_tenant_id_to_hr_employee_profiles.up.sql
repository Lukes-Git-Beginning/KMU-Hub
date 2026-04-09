ALTER TABLE hr_employee_profiles ADD COLUMN tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';
CREATE INDEX idx_hr_employee_profiles_tenant ON hr_employee_profiles(tenant_id);

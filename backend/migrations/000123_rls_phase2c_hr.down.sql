-- Reverse of 000123 — drop tenant_isolation policy and disable RLS
-- on the 8 hr_* tables, in reverse order of activation.

SET LOCAL row_security = off;

BEGIN;

-- Custom-policy tables (hr_document_categories, hr_leave_types)
DROP POLICY IF EXISTS tenant_isolation ON hr_document_categories;
ALTER TABLE hr_document_categories DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON hr_leave_types;
ALTER TABLE hr_leave_types DISABLE ROW LEVEL SECURITY;

-- Standard-policy tables (reverse order)
DROP POLICY IF EXISTS tenant_isolation ON hr_employee_profiles;
ALTER TABLE hr_employee_profiles DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON hr_employee_documents;
ALTER TABLE hr_employee_documents DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON hr_work_time_entries;
ALTER TABLE hr_work_time_entries DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON hr_leave_balances;
ALTER TABLE hr_leave_balances DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON hr_leave_requests;
ALTER TABLE hr_leave_requests DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON hr_company_settings;
ALTER TABLE hr_company_settings DISABLE ROW LEVEL SECURITY;

COMMIT;

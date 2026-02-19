-- Migration 000046 DOWN: Drop HR tables in reverse order

DROP TABLE IF EXISTS hr_employee_documents;
DROP TABLE IF EXISTS hr_document_categories;
DROP TABLE IF EXISTS hr_break_entries;
DROP TABLE IF EXISTS hr_work_time_entries;
DROP TABLE IF EXISTS hr_leave_balances;
DROP TABLE IF EXISTS hr_leave_requests;
DROP TABLE IF EXISTS hr_leave_types;
DROP TABLE IF EXISTS hr_employee_profiles;
DROP TABLE IF EXISTS hr_company_settings;

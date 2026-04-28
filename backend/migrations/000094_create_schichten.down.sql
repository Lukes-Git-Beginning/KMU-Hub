-- Sprint 2 W2.A Schichten — rollback

DROP INDEX IF EXISTS idx_shift_templates_tenant_id;
DROP TABLE IF EXISTS shift_templates;

DROP INDEX IF EXISTS idx_shift_assignments_employee_id;
DROP INDEX IF EXISTS idx_shift_assignments_shift_id;
DROP INDEX IF EXISTS idx_shift_assignments_tenant_id;
DROP TABLE IF EXISTS shift_assignments;

DROP INDEX IF EXISTS idx_shifts_published;
DROP INDEX IF EXISTS idx_shifts_tenant_period;
DROP INDEX IF EXISTS idx_shifts_tenant_status;
DROP INDEX IF EXISTS idx_shifts_tenant_id;
DROP TABLE IF EXISTS shifts;

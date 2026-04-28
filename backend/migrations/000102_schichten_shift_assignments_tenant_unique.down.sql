-- Rollback 000102

ALTER TABLE shift_assignments
    DROP CONSTRAINT IF EXISTS uq_shift_assignments_tenant;

ALTER TABLE shift_assignments
    ADD CONSTRAINT shift_assignments_shift_id_employee_id_key
    UNIQUE (shift_id, employee_id);

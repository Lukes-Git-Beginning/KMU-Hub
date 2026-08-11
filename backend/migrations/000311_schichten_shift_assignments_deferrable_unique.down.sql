ALTER TABLE shift_assignments
    DROP CONSTRAINT uq_shift_assignments_tenant;

ALTER TABLE shift_assignments
    ADD CONSTRAINT uq_shift_assignments_tenant
    UNIQUE (tenant_id, shift_id, employee_id);

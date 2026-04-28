-- Migration 000102: Fix shift_assignments uniqueness to include tenant_id (Bug #7)
-- The existing UNIQUE (shift_id, employee_id) does not include tenant_id,
-- allowing cross-tenant collisions if two tenants share shift UUIDs (unlikely
-- but architecturally unsafe once Option-B multi-tenant is active).

-- Drop the old constraint (no tenant_id)
ALTER TABLE shift_assignments
    DROP CONSTRAINT IF EXISTS shift_assignments_shift_id_employee_id_key;

-- Add the correct tenant-scoped unique constraint
ALTER TABLE shift_assignments
    ADD CONSTRAINT uq_shift_assignments_tenant
    UNIQUE (tenant_id, shift_id, employee_id);

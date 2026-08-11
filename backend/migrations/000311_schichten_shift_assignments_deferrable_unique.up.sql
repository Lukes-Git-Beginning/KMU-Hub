-- Migration 000311: Make uq_shift_assignments_tenant DEFERRABLE.
--
-- SwapAssignmentsForRequest swaps two shift_assignments rows' employee_id
-- values inside one transaction. With a NOT DEFERRABLE (the default)
-- constraint, Postgres validates uq_shift_assignments_tenant right after
-- each UPDATE, so the moment the requester's row is set to the partner's
-- employee_id -- while the partner's own row still carries that same
-- value -- the statement fails with SQLSTATE 23505, even though the swap
-- is fully resolved by the time the transaction would commit.
--
-- DEFERRABLE INITIALLY IMMEDIATE keeps every other write path unchanged
-- (the check still runs immediately unless a transaction opts in via
-- SET CONSTRAINTS ... DEFERRED, which only SwapAssignmentsForRequest does)
-- while letting the swap defer the check to COMMIT, when both rows hold
-- their final, non-colliding values.

ALTER TABLE shift_assignments
    DROP CONSTRAINT uq_shift_assignments_tenant;

ALTER TABLE shift_assignments
    ADD CONSTRAINT uq_shift_assignments_tenant
    UNIQUE (tenant_id, shift_id, employee_id)
    DEFERRABLE INITIALLY IMMEDIATE;

-- Migration 000160: shift_swap_requests
-- Allows employees to request swapping shift assignments with each other.
-- Status lifecycle: pending → approved | rejected.
-- Approved swaps atomically exchange the two assignments in the service layer.

CREATE TABLE shift_swap_requests (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID        NOT NULL,
    shift_id                 UUID        NOT NULL REFERENCES shifts (id) ON DELETE CASCADE,
    assignment_id            UUID        NOT NULL REFERENCES shift_assignments (id) ON DELETE CASCADE,
    requested_by_employee_id UUID        NOT NULL,
    swap_with_employee_id    UUID        NOT NULL,
    status                   TEXT        NOT NULL DEFAULT 'pending'
                                         CHECK (status IN ('pending', 'approved', 'rejected')),
    reason                   TEXT        NOT NULL DEFAULT '',
    idempotency_key          TEXT        UNIQUE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT swap_employees_differ
        CHECK (requested_by_employee_id <> swap_with_employee_id)
);

CREATE INDEX idx_swap_requests_tenant_id  ON shift_swap_requests (tenant_id);
CREATE INDEX idx_swap_requests_shift_id   ON shift_swap_requests (tenant_id, shift_id);
CREATE INDEX idx_swap_requests_status     ON shift_swap_requests (tenant_id, status);

CALL enable_tenant_rls('shift_swap_requests');

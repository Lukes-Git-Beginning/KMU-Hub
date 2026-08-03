-- HR self-service change requests (backlog unit g-hr-change-requests, Lauf 4).
--
-- An employee who may not edit their own master data proposes a change instead;
-- somebody holding team:data_personal:edit approves or rejects it. One row per
-- proposal, and the row keeps the whole story (old value, new value, decision,
-- who decided) so the personnel file has an audit trail of every change that
-- went through the flow.
--
-- old_value is stored, not resolved at approval time: it is what the employee
-- SAW when proposing. If HR edits the field in the meantime, the approver has
-- to see that the proposal was made against a different starting point.

CREATE TABLE IF NOT EXISTS hr_profile_change_requests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    drawer       VARCHAR(16) NOT NULL
                 CONSTRAINT chk_hr_change_request_drawer CHECK (drawer IN ('personal', 'job')),
    field        VARCHAR(64) NOT NULL,
    field_label  VARCHAR(120) NOT NULL,
    old_value    TEXT NOT NULL DEFAULT '',
    new_value    TEXT NOT NULL,
    status       VARCHAR(16) NOT NULL DEFAULT 'pending'
                 CONSTRAINT chk_hr_change_request_status
                 CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    reason       TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at   TIMESTAMPTZ,
    decided_by   UUID REFERENCES users(id) ON DELETE SET NULL
);

-- The 409 on a second open proposal for the same field is enforced here, not
-- only in the service: two concurrent POSTs both pass a SELECT-then-INSERT
-- check and would otherwise leave two pending rows that contradict each other.
CREATE UNIQUE INDEX IF NOT EXISTS uq_hr_change_requests_pending_field
    ON hr_profile_change_requests (tenant_id, user_id, field)
    WHERE status = 'pending';

-- The inbox reads "pending, newest first"; the self-service view reads one
-- user's own rows.
CREATE INDEX IF NOT EXISTS idx_hr_change_requests_tenant_status
    ON hr_profile_change_requests (tenant_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_hr_change_requests_user
    ON hr_profile_change_requests (tenant_id, user_id, created_at DESC);

CALL enable_tenant_rls('hr_profile_change_requests');

-- No permission seed: the two guards this flow uses (team:self:propose for
-- submitting and cancelling, team:data_personal:edit for deciding) both exist
-- in the catalogue since migration 000256 and are already granted to the
-- presets that should hold them. Adding them again would be a no-op.

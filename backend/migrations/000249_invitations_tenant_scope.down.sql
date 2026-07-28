-- Rollback of migration 000249.
--
-- Recreating the global partial unique index fails if two tenants each hold a
-- pending invitation for the same address — data that could only exist after
-- the up migration. That is deliberate: the rollback stops rather than delete
-- one of the two invitations behind the operator's back.

BEGIN;

ALTER TABLE tenants DROP COLUMN IF EXISTS seat_limit;

DROP POLICY IF EXISTS tenant_isolation ON invitations;
ALTER TABLE invitations NO FORCE ROW LEVEL SECURITY;
ALTER TABLE invitations DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_invitations_tenant_created;
DROP INDEX IF EXISTS idx_invitations_tenant_email_pending;
CREATE UNIQUE INDEX idx_invitations_email_pending
    ON invitations (email) WHERE accepted_at IS NULL;

ALTER TABLE invitations DROP CONSTRAINT IF EXISTS fk_invitations_tenant;
ALTER TABLE invitations DROP COLUMN IF EXISTS tenant_id;

COMMIT;

-- Migration 000249: give invitations a tenant, and tenants a seat limit.
--
-- invitations predates the tenant model (migration 000004) and never got a
-- tenant_id. Three things follow from that, and all three are wrong:
--   * the pending list is global — an admin of tenant A sees the invitations
--     tenant B has open, including the invited addresses,
--   * the unique index on (email) WHERE accepted_at IS NULL is global, so
--     tenant B cannot invite an address tenant A happens to have pending,
--   * accepting an invitation put the new user into the default tenant
--     regardless of who invited them.
-- The column, the RLS policy and the per-tenant uniqueness close all three.
--
-- seat_limit lives on tenants because a seat is a property of the booked plan,
-- not of a single invitation. NULL means unlimited, which is what every tenant
-- gets here: this migration only creates the place where the licence service
-- (p3-admin-billing-license) will write the number. Enforcement is already in
-- the invite path, so the limit takes effect the moment a value is written.

BEGIN;

ALTER TABLE invitations ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Backfill from the inviter: an invitation belongs to the tenant of the admin
-- who issued it. Rows whose creator is gone fall back to the default tenant —
-- created_by is ON DELETE CASCADE, so this only ever catches rows written
-- before that constraint existed.
UPDATE invitations i
SET tenant_id = u.tenant_id
FROM users u
WHERE i.created_by = u.id
  AND i.tenant_id IS NULL;

UPDATE invitations
SET tenant_id = '00000000-0000-0000-0000-000000000001'
WHERE tenant_id IS NULL;

ALTER TABLE invitations ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE invitations
    ADD CONSTRAINT fk_invitations_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants (id) ON DELETE CASCADE;

-- Pending uniqueness is per tenant. Dropped and recreated rather than widened,
-- because a partial unique index cannot be altered in place.
DROP INDEX IF EXISTS idx_invitations_email_pending;
CREATE UNIQUE INDEX idx_invitations_tenant_email_pending
    ON invitations (tenant_id, email) WHERE accepted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_invitations_tenant_created
    ON invitations (tenant_id, created_at DESC);

ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;
ALTER TABLE invitations FORCE ROW LEVEL SECURITY;
-- The accept path runs before a JWT exists and therefore under the system
-- context (sysctx.With) — is_system_context() is what lets it read the
-- invitation by token hash at all.
CREATE POLICY tenant_isolation ON invitations
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS seat_limit INTEGER;

COMMENT ON COLUMN tenants.seat_limit IS
    'Booked seats. NULL means unlimited. An active user and a pending, unexpired invitation each occupy one seat.';

COMMIT;

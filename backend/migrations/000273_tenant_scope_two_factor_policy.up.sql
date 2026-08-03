-- ============================================================================
-- First of the five tables found by backlog unit g-rls-tenant-scoped-admin-writes:
-- data mutated through a tenant-scoped admin route that carries no tenant_id at
-- all, so the guard checks "is an admin" but never "an admin of which tenant".
--
-- two_factor_policy is the most damaging of the five and therefore goes first.
-- PUT /auth/2fa/policy is guarded by RequireRole("admin") (route_auth.go:90) and
-- upserts on a globally unique role_name, so any tenant's administrator could
-- switch off two-factor enforcement — or shorten its grace period — for every
-- other tenant on the installation at the same time. Nothing in the read path
-- noticed: Check2FAEnforcement treats a missing policy as "not enforced"
-- (totp.go:280, fail-open), so the victim tenant sees no error, just silently
-- unenforced 2FA.
--
-- The backfill replicates each existing global row onto every tenant, which is
-- the only lossless reading of the old data: the row applied to all of them, so
-- all of them keep exactly the policy that governed them before this migration.
-- updated_by is kept only for the tenant the editing user actually belongs to —
-- carrying a foreign tenant's user id into another tenant's row would leak an
-- account reference across the very boundary this migration draws. Production
-- is single-tenant today, so the replication is 1:1 there.
--
-- No provisioning default is seeded for new tenants, unlike what the backlog
-- unit assumed for the group as a whole. It would be dead rows: an absent
-- policy and a row with the column defaults (enforced=false,
-- grace_period_days=14) mean the identical thing to Check2FAEnforcement, and
-- UpdateTwoFactorPolicy inserts on first write. The other four tables in the
-- unit read with LIMIT 1 and genuinely need a seeded row — this one does not.
-- ============================================================================

ALTER TABLE two_factor_policy
    ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

-- Dropped before the backfill, not after: the replication writes one row per
-- tenant per role and would collide with a unique index on role_name alone.
DROP INDEX idx_two_factor_policy_role;

INSERT INTO two_factor_policy (id, tenant_id, role_name, enforced, grace_period_days, updated_at, updated_by)
SELECT gen_random_uuid(), t.id, p.role_name, p.enforced, p.grace_period_days, p.updated_at,
       CASE WHEN u.tenant_id = t.id THEN p.updated_by END
FROM two_factor_policy p
CROSS JOIN tenants t
LEFT JOIN users u ON u.id = p.updated_by
WHERE p.tenant_id IS NULL;

DELETE FROM two_factor_policy WHERE tenant_id IS NULL;

ALTER TABLE two_factor_policy ALTER COLUMN tenant_id SET NOT NULL;

-- tenant_id leads the index, so it also serves the policy's filter and the
-- ListTwoFactorPolicies scan; a separate index on tenant_id would be redundant.
CREATE UNIQUE INDEX idx_two_factor_policy_tenant_role ON two_factor_policy (tenant_id, role_name);

CALL enable_tenant_rls('two_factor_policy');

-- ============================================================================
-- Second slice of backlog unit g-rls-tenant-scoped-admin-writes, following
-- 000273 (two_factor_policy): tables whose rows are mutated through a
-- tenant-scoped admin route while carrying no tenant_id, so the guard answers
-- "is this an admin" but never "an admin of which tenant".
--
-- presence_config held exactly one row for the whole installation and
-- UpdateConfig ran `UPDATE presence_config SET away_timeout_seconds=$1` with no
-- WHERE at all (postgres_config_repository.go:39). PUT /presence/config is
-- guarded by settings:write, so any tenant's administrator moved the away
-- timeout for every other tenant on the installation.
--
-- dashboard_defaults was unique on role alone, and UpsertDefaultLayout upserted
-- on that key (dashboard_repository.go:60), so PUT /admin/dashboard/defaults/
-- {role} overwrote the role preset of every tenant at once.
--
-- Both backfills replicate the existing global rows onto every tenant, the same
-- lossless reading 000273 used: the row governed all of them, so all of them
-- keep the setting they were under. Production is single-tenant today, so the
-- replication is 1:1 there.
--
-- No provisioning seed is added for new tenants. Both read paths fall back to a
-- code default when the row is missing — presence to DefaultAwayTimeoutSeconds
-- (models.go:49), the dashboard to hardcodedDefaultLayout (dashboard_service.go:
-- 24) — and both write paths upsert, so the row appears on first write. A
-- seeded row would only duplicate a default that already exists in code.
-- ============================================================================

-- ---------------------------------------------------------------------------
-- presence_config
-- ---------------------------------------------------------------------------

ALTER TABLE presence_config
    ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

-- The table never had a unique key, only the id primary key, so in principle
-- several global rows could exist. They would all carry the same value (every
-- write was an unfiltered UPDATE), but replicating each of them would collide
-- with the tenant-unique index below. Take the most recently edited one.
INSERT INTO presence_config (id, tenant_id, away_timeout_seconds, updated_at, updated_by)
SELECT gen_random_uuid(), t.id, p.away_timeout_seconds, p.updated_at,
       CASE WHEN u.tenant_id = t.id THEN p.updated_by END
FROM (
    SELECT * FROM presence_config
     WHERE tenant_id IS NULL
     ORDER BY updated_at DESC, id
     LIMIT 1
) p
CROSS JOIN tenants t
LEFT JOIN users u ON u.id = p.updated_by;

DELETE FROM presence_config WHERE tenant_id IS NULL;

ALTER TABLE presence_config ALTER COLUMN tenant_id SET NOT NULL;

-- One config row per tenant. Also the conflict target UpdateConfig upserts on.
CREATE UNIQUE INDEX idx_presence_config_tenant ON presence_config (tenant_id);

CALL enable_tenant_rls('presence_config');

-- ---------------------------------------------------------------------------
-- dashboard_defaults
-- ---------------------------------------------------------------------------

ALTER TABLE dashboard_defaults
    ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;

-- Dropped before the backfill: the replication writes one row per tenant per
-- role and would collide with a unique constraint on role alone.
ALTER TABLE dashboard_defaults DROP CONSTRAINT dashboard_defaults_role_key;
DROP INDEX idx_dashboard_defaults_role;

INSERT INTO dashboard_defaults (id, tenant_id, role, layout, active_widgets, created_at, updated_at)
SELECT gen_random_uuid(), t.id, d.role, d.layout, d.active_widgets, d.created_at, d.updated_at
FROM dashboard_defaults d
CROSS JOIN tenants t
WHERE d.tenant_id IS NULL;

DELETE FROM dashboard_defaults WHERE tenant_id IS NULL;

ALTER TABLE dashboard_defaults ALTER COLUMN tenant_id SET NOT NULL;

-- tenant_id leads, so this index also serves the policy filter and the
-- per-role lookup the old idx_dashboard_defaults_role covered.
CREATE UNIQUE INDEX idx_dashboard_defaults_tenant_role ON dashboard_defaults (tenant_id, role);

CALL enable_tenant_rls('dashboard_defaults');

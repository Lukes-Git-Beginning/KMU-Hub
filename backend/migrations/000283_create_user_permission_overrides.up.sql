-- Per-user permission overrides (backlog unit g-rbac-user-overrides-model,
-- RBAC R-6). One row per (user, capability key) that deviates from what the
-- user's roles grant: `allow` sets or raises the key, `deny` takes it away
-- regardless of how many roles grant it. A key WITHOUT a row here keeps
-- following the roles live — that is the whole point of the feature over
-- cloning a role for one person (R6-USER-OVERRIDES-BRIEFING §3).
--
-- This migration only creates the storage. The resolution step that folds
-- these rows into GET /auth/me/permissions is a separate unit; until it lands
-- the rows are written and read back by the editor but change no gate.
--
-- permission_key is the catalogue's `permissions.name` string rather than a
-- permission_id FK like role_permissions uses: the frontend contract
-- (rbac-types.ts, UserOverrides) is a flat `Record<key, {mode, scope}>`, and
-- every read of this table hands the keys straight back out. The FK keeps the
-- referential guarantee anyway — permissions.name carries a unique index, and
-- a key retired from the catalogue takes its overrides with it instead of
-- leaving rows that name a right nothing checks for.

CREATE TABLE IF NOT EXISTS user_permission_overrides (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_key VARCHAR(120) NOT NULL
                   REFERENCES permissions(name) ON UPDATE CASCADE ON DELETE CASCADE,
    mode           VARCHAR(8) NOT NULL
                   CONSTRAINT chk_user_permission_override_mode CHECK (mode IN ('allow', 'deny')),
    -- Stored for both modes, meaningful only for `allow`. The frontend type
    -- carries it either way ("ignored (but stored) for deny"), and keeping the
    -- column NOT NULL saves every reader a null check for a value that has a
    -- natural default.
    scope          VARCHAR(8) NOT NULL DEFAULT 'all'
                   CONSTRAINT chk_user_permission_override_scope CHECK (scope IN ('own', 'team', 'all')),
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One override per key per user. The PUT replaces the whole map inside a
-- transaction, so this index is not what the happy path relies on — it is what
-- keeps two concurrent PUTs from leaving two contradictory rows for one key.
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_permission_overrides_key
    ON user_permission_overrides (tenant_id, user_id, permission_key);

-- The two reads: one account's map (editor), and "who deviates at all" plus
-- the last-admin guardrail, which asks for a deny on a specific key.
CREATE INDEX IF NOT EXISTS idx_user_permission_overrides_user
    ON user_permission_overrides (tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_user_permission_overrides_key
    ON user_permission_overrides (tenant_id, permission_key, mode);

CALL enable_tenant_rls('user_permission_overrides');

-- No permission seed: admin:user_override:manage — the key the three routes
-- guard with — is in the catalogue since migration 000256 and granted to the
-- `admin` preset there (and deliberately to no other preset: hr_admin assigns
-- roles without being allowed to fine-tune individuals, R6 briefing §0.3).
-- Re-seeding it would be a no-op.

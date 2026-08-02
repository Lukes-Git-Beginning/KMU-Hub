-- ============================================================================
-- Part of the ADR-006 allowlist audit (backlog unit g-rls-allowlist-audit):
-- eleven tables without tenant_id/RLS were checked against the four-entry
-- allowlist in docs/ARCHITECTURE.md. Two of them are genuine gaps closed here.
-- The other nine are handled in the same commit's follow-up: five are pure
-- catalog/seed data added to the allowlist, four are a distinct bug (a
-- tenant-scoped admin guard mutating data with no tenant_id at all) split into
-- its own backlog unit rather than folded into this migration.
--
-- refresh_tokens: user_id NOT NULL, FK to users(id) ON DELETE CASCADE, zero
-- orphans on the local database — the backfill is a plain join, unlike events'
-- three-step guess. The table already has a proven sibling: password_reset_tokens
-- (migration predates this audit) carries tenant_id and is written under
-- sysctx.With() from RequestPasswordReset/ConfirmPasswordReset, because the
-- token is presented before a session exists and no tenant is on the context
-- yet. auth/service.go's RefreshToken and Logout already wrap their whole body
-- in sysctx.With() for the identical reason (comment: "Pre-JWT path ... RLS
-- bypass needed for the user_sessions + users lookups") — user_sessions
-- already carries tenant_id itself. refresh_tokens was the one caller in that
-- same pre-auth path left without a tenant_id, and without a policy anyone
-- holding a valid connection could read or revoke any tenant's tokens by id,
-- not just by hash. No Go call site changes need a new sysctx.With(): the two
-- pre-auth entry points already have it, and the two authenticated entry points
-- (session.go's TerminateSession/TerminateAllSessions) already verify the
-- target session belongs to the caller before touching its refresh token, so
-- caller tenant and row tenant already agree.
--
-- plugin_permissions: per-installation grants, no tenant_id of its own, but
-- installation_id is NOT NULL with a FK to plugin_installations, which has
-- carried tenant_id + RLS since its own migration. Same shape as the CRM
-- custom-field-value tables in 000270: reached only through the parent id,
-- which leads the (installation_id, permission) unique index, so the join
-- costs one indexed lookup on an already-narrow set. enable_tenant_rls_via_join()
-- over a NOT NULL column beats adding and backfilling a second tenant_id here.
-- ============================================================================

ALTER TABLE refresh_tokens ADD COLUMN tenant_id UUID;

UPDATE refresh_tokens rt
SET tenant_id = u.tenant_id
FROM users u
WHERE u.id = rt.user_id;

ALTER TABLE refresh_tokens ALTER COLUMN tenant_id SET NOT NULL;

CREATE INDEX idx_refresh_tokens_tenant_id ON refresh_tokens(tenant_id);

CALL enable_tenant_rls('refresh_tokens');

CALL enable_tenant_rls_via_join('plugin_permissions', 'plugin_installations', 'installation_id');

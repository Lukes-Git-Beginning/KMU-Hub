-- Sprint 4 Welle 2 — RLS-Aktivierung Phase 1a auf 22 Pilot-0-Tabellen.
-- 2026-05-11
--
-- Builds on 000118 (helper functions + enable_tenant_rls procedure) and 000119
-- (tenant_id backfill on the four child tables). Activates ROW LEVEL SECURITY
-- with the standard tenant_isolation policy on 19 tables, plus three custom
-- policies for the special cases tenants (no tenant_id column, root table),
-- users (defense-in-depth USING with self-row read), and idempotency_keys
-- (composite PK already covers tenant scoping but we still need RLS).
--
-- Pre-flight: every targeted table already carries a NOT NULL tenant_id
-- column (verified against migrations 000106/000111/000114/000115/000119).
-- The exception is `tenants` itself, where the row's `id` IS the tenant.
--
-- Migration runs with row_security disabled inside the migration tx so the
-- ALTER TABLE statements themselves are not subject to a (not yet existing)
-- policy. Same pattern as 000118.

SET LOCAL row_security = off;

BEGIN;

-- ============================================================================
-- CRM-Core (7) — Standard tenant_isolation policy via enable_tenant_rls().
-- ============================================================================
CALL enable_tenant_rls('contacts');
CALL enable_tenant_rls('companies');
CALL enable_tenant_rls('deals');
CALL enable_tenant_rls('activities');
CALL enable_tenant_rls('pipeline_stages');
CALL enable_tenant_rls('deal_stage_history');
CALL enable_tenant_rls('tasks');

-- ============================================================================
-- Dialer (7) — Standard policy. Children backfilled NOT NULL in 000119.
-- ============================================================================
CALL enable_tenant_rls('dialer_campaigns');
CALL enable_tenant_rls('dialer_campaign_contacts');
CALL enable_tenant_rls('dialer_call_sessions');
CALL enable_tenant_rls('dialer_call_outcomes');
CALL enable_tenant_rls('dialer_call_events');
CALL enable_tenant_rls('dialer_agent_status_log');
CALL enable_tenant_rls('consent_records');

-- ============================================================================
-- Recording / Audit (3) — Standard policy.
-- ============================================================================
CALL enable_tenant_rls('recordings');
CALL enable_tenant_rls('recording_consents');
CALL enable_tenant_rls('audit_log');

-- ============================================================================
-- Idempotency (1) — Composite PK (tenant_id, key) since 000108. The standard
-- policy filters on the tenant_id column, independent of the PK shape, so the
-- procedure works unmodified. Cross-tenant keys remain insertable (separate
-- rows under the composite PK) but only readable to the owning tenant.
-- ============================================================================
CALL enable_tenant_rls('idempotency_keys');

-- ============================================================================
-- user_sessions (1) — Standard policy. Auth-Service Pre-JWT methods (Login,
-- Register, RefreshToken, Logout, CompleteTwoFactorLogin, AcceptInvitation)
-- wrap their ctx with sysctx.With (Welle 2a), so session lookups bypass the
-- tenant filter via is_system_context(). Post-JWT calls flow through the
-- BeforeAcquire pool hook with app.tenant_id stamped from middleware.
-- ============================================================================
CALL enable_tenant_rls('user_sessions');

-- ============================================================================
-- tenants (1) — CUSTOM POLICY. Root table, no tenant_id column. A user may
-- read only their own tenant; only system-context (migrations, provisioning
-- workers) may write. Sprint 5 may relax WITH CHECK once SaaS self-service
-- onboarding lands.
-- ============================================================================
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_self_isolation ON tenants
    USING (id = current_tenant_id() OR is_system_context())
    WITH CHECK (is_system_context());

-- ============================================================================
-- users (1) — CUSTOM POLICY. Defense-in-depth: a user may read their own row
-- even if app.tenant_id is somehow missing (e.g. JWT-reissue race during a
-- token refresh) so the UI does not lock up on stale tokens. WITH CHECK is
-- still strict — no cross-tenant user moves via the application layer.
-- ============================================================================
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE POLICY user_isolation ON users
    USING (
        tenant_id = current_tenant_id()
        OR id = current_user_id()
        OR is_system_context()
    )
    WITH CHECK (
        tenant_id = current_tenant_id()
        OR is_system_context()
    );

COMMIT;

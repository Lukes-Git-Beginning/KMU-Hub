-- ============================================================================
-- user_module_grants — which modules a user has ACCESS to within a tenant.
-- Distinct from tenant_module_leads (who LEADS a module). Admins grant/revoke
-- access per user in the Team module (ModuleAssignmentTab). R3-E6.
--
-- last_active_at is reserved for a future activity-tracking pipeline and stays
-- NULL until that lands; the UI renders NULL as "never"/"—". Populating it is a
-- tracked follow-up, not part of this migration.
-- ============================================================================

CREATE TABLE IF NOT EXISTS user_module_grants (
    tenant_id      UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id        UUID        NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    module_id      TEXT        NOT NULL,
    granted_by     UUID                 REFERENCES users(id)   ON DELETE SET NULL,
    granted_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ,

    CONSTRAINT user_module_grants_pk PRIMARY KEY (tenant_id, user_id, module_id),
    CONSTRAINT user_module_grants_module_id_nonempty CHECK (module_id <> '')
);

CREATE INDEX IF NOT EXISTS idx_user_module_grants_tenant_id ON user_module_grants (tenant_id);
CREATE INDEX IF NOT EXISTS idx_user_module_grants_user_id   ON user_module_grants (tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_user_module_grants_module_id ON user_module_grants (tenant_id, module_id);

-- Tenant isolation: new tenant_id table created under enforced RLS (ADR-006).
CALL enable_tenant_rls('user_module_grants');

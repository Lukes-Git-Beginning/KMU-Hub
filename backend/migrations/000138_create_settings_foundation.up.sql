-- Migration 000138 — Settings Foundation (3-level scope model)
--
-- Implements the tenant-default → Modul-Leiter-override (tenant-wide) → user-override (personal)
-- hierarchy described in .planning/backend-gaps.md §Settings-Fundament.
--
-- Three new tables:
--   tenant_module_leads  — who is Modul-Leiter for which module (admin delegates)
--   tenant_settings      — tenant-wide defaults per module+key (module-lead/admin write)
--   user_settings        — personal user overrides per module+key (own user writes)
--
-- Permission seeds:
--   module-leads:read/write  — for the tenant_module_leads endpoints
--   settings:read/write      — for settings get/put endpoints
--
-- Resolve order (serverseitig): user_settings > tenant_settings > not-set.
-- tenant_id NOT NULL on all three tables (Option-B, pre-RLS-ready).

BEGIN;

-- ============================================================================
-- tenant_module_leads
-- Who is Modul-Leiter for which module in a tenant. Admin sets this per user
-- in the Team module (MemberDetailPanel → "Erweiterte Moduleinstellungen").
-- Admins are implicitly lead for all modules — enforced in service logic, not here.
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenant_module_leads (
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    module_id  TEXT        NOT NULL,
    granted_by UUID                 REFERENCES users(id)   ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_module_leads_pk PRIMARY KEY (tenant_id, user_id, module_id),
    CONSTRAINT tenant_module_leads_module_id_nonempty CHECK (module_id <> '')
);

CREATE INDEX IF NOT EXISTS idx_tenant_module_leads_tenant_id ON tenant_module_leads (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_module_leads_user_id   ON tenant_module_leads (tenant_id, user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_module_leads_module_id ON tenant_module_leads (tenant_id, module_id);

-- ============================================================================
-- tenant_settings
-- Tenant-wide defaults for a module+key pair. Only module-leads or admins may
-- write these. value is JSONB to accommodate arbitrary shapes per key
-- (e.g. {"workStartHour":8} or "08:00" or [{"id":"..."},...]).
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenant_settings (
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_id  TEXT        NOT NULL,
    key        TEXT        NOT NULL,
    value      JSONB       NOT NULL,
    updated_by UUID                 REFERENCES users(id)   ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_settings_pk PRIMARY KEY (tenant_id, module_id, key),
    CONSTRAINT tenant_settings_module_id_nonempty CHECK (module_id <> ''),
    CONSTRAINT tenant_settings_key_nonempty       CHECK (key <> '')
);

CREATE INDEX IF NOT EXISTS idx_tenant_settings_tenant_module ON tenant_settings (tenant_id, module_id);

-- ============================================================================
-- user_settings
-- Personal overrides per user/module/key. User writes only their own.
-- FK on users CASCADE-deletes settings when a user is removed (GDPR).
-- ============================================================================

CREATE TABLE IF NOT EXISTS user_settings (
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    module_id  TEXT        NOT NULL,
    key        TEXT        NOT NULL,
    value      JSONB       NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT user_settings_pk PRIMARY KEY (tenant_id, user_id, module_id, key),
    CONSTRAINT user_settings_module_id_nonempty CHECK (module_id <> ''),
    CONSTRAINT user_settings_key_nonempty       CHECK (key <> '')
);

CREATE INDEX IF NOT EXISTS idx_user_settings_user_module ON user_settings (tenant_id, user_id, module_id);

-- ============================================================================
-- Permission seeds
-- Without these rows RequirePermission("settings"|"module-leads", ...) returns
-- 403 for everyone including admins. Pattern follows 000131/000136.
-- ============================================================================

INSERT INTO permissions (name, resource, action) VALUES
    ('module-leads:read',  'module-leads', 'read'),
    ('module-leads:write', 'module-leads', 'write'),
    ('settings:read',      'settings',     'read'),
    ('settings:write',     'settings',     'write')
ON CONFLICT (name) DO NOTHING;

-- module-leads:read — any authenticated user (list leads for a module they belong to)
-- module-leads:write — admin only (grant/revoke module-lead role)
-- settings:read — all roles (everyone reads resolved settings)
-- settings:write — all roles (user-scope writes; service enforces tenant-scope restriction)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN ('module-leads:read', 'module-leads:write', 'settings:read', 'settings:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name IN ('manager', 'member')
  AND p.name IN ('module-leads:read', 'settings:read', 'settings:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

COMMIT;

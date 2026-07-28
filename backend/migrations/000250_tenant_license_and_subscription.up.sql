-- Migration 000250: tenant subscription fields and tenant-wide module activation.
--
-- Two things the admin hub reads today from a mock:
--   * the booked plan (Cosmi/Orbit), support tier, billing period and status —
--     properties of the contract, so they live on the tenant row next to the
--     seat_limit that 000249 put there,
--   * which modules the tenant has activated, which is a per-tenant row and
--     therefore its own table under RLS.
--
-- Activation is bookkeeping, not enforcement: nothing in this migration blocks a
-- request to a deactivated module. The deployment-wide modules.* feature flags
-- keep doing that job, and the gateway masks an activation the flag does not
-- cover. Turning activation into an access check is a separate unit, because it
-- can lock a tenant out of a module it is using.

BEGIN;

-- ── Subscription on the tenant row ──────────────────────────────────────────

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS plan_type           TEXT NOT NULL DEFAULT 'cosmi';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS support_tier        TEXT NOT NULL DEFAULT 'standard';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS subscription_status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS billing_period_end  DATE;

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_plan_type;
ALTER TABLE tenants ADD  CONSTRAINT chk_tenants_plan_type
    CHECK (plan_type IN ('cosmi', 'orbit'));

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_support_tier;
ALTER TABLE tenants ADD  CONSTRAINT chk_tenants_support_tier
    CHECK (support_tier IN ('standard', 'priority', 'enterprise'));

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_subscription_status;
ALTER TABLE tenants ADD  CONSTRAINT chk_tenants_subscription_status
    CHECK (subscription_status IN ('active', 'past_due', 'paused'));

COMMENT ON COLUMN tenants.plan_type IS
    'Booked product: cosmi (SaaS) or orbit (self-hosted). Written by the licence process, read by the admin hub.';
COMMENT ON COLUMN tenants.billing_period_end IS
    'End of the current billing period. NULL while no period has been agreed.';

-- ── Tenant-wide module activation ───────────────────────────────────────────

CREATE TABLE IF NOT EXISTS tenant_module_activations (
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_id  TEXT        NOT NULL,
    active     BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_by UUID                 REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_module_activations_pk PRIMARY KEY (tenant_id, module_id),
    CONSTRAINT tenant_module_activations_module_id_nonempty CHECK (module_id <> '')
);

-- Tenant isolation: new tenant_id table created under enforced RLS (ADR-006).
CALL enable_tenant_rls('tenant_module_activations');

-- Backfill every existing tenant with every catalogue module active. A missing
-- row reads as inactive in the service, which is the right default for a tenant
-- provisioned later — but flipping existing tenants to "nothing booked" would
-- misrepresent what they are using today. The id list mirrors ModuleCatalog in
-- internal/settings/modules.go; a module added there after this migration is
-- inactive until someone activates it, which is the intended direction.
INSERT INTO tenant_module_activations (tenant_id, module_id, active)
SELECT t.id, m.module_id, TRUE
FROM tenants t
CROSS JOIN (VALUES
    ('crm'), ('tasks'), ('finance'), ('calendar'), ('documents'),
    ('chat'), ('meetings'), ('mail'), ('dialer'),
    ('team'), ('zeiterfassung'), ('schichten'),
    ('projects'), ('vertraege'), ('helpdesk'), ('inventar'),
    ('einkauf'), ('fuhrpark'), ('produktion'), ('vermietung'),
    ('berichte'), ('formulare'), ('wiki'), ('rapporte')
) AS m(module_id)
ON CONFLICT (tenant_id, module_id) DO NOTHING;

-- ── Permissions for the two licence endpoints ───────────────────────────────
-- Without these rows RequirePermission("license", ...) returns 403 for everyone
-- including admins. Pattern follows 000221.

INSERT INTO permissions (name, resource, action) VALUES
    ('license:read',  'license', 'read'),
    ('license:write', 'license', 'write')
ON CONFLICT (name) DO NOTHING;

-- license:read  — admin + manager (see the booked plan and active modules)
-- license:write — admin only (activate/deactivate a module tenant-wide)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN ('license:read', 'license:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.name = 'license:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

COMMIT;

-- Rollback of migration 000250.

BEGIN;

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE name IN ('license:read', 'license:write')
);

DELETE FROM permissions WHERE name IN ('license:read', 'license:write');

DROP TABLE IF EXISTS tenant_module_activations;

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_subscription_status;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_support_tier;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS chk_tenants_plan_type;

ALTER TABLE tenants DROP COLUMN IF EXISTS billing_period_end;
ALTER TABLE tenants DROP COLUMN IF EXISTS subscription_status;
ALTER TABLE tenants DROP COLUMN IF EXISTS support_tier;
ALTER TABLE tenants DROP COLUMN IF EXISTS plan_type;

COMMIT;

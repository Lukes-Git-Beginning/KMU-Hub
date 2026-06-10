-- Rollback 000138 — drop settings foundation tables and permission seeds

BEGIN;

-- Remove permission grants first (FK references permissions)
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions
    WHERE name IN ('module-leads:read', 'module-leads:write', 'settings:read', 'settings:write')
);

DELETE FROM permissions
WHERE name IN ('module-leads:read', 'module-leads:write', 'settings:read', 'settings:write');

-- Drop tables (user_settings first because of no cross-deps, tenant_module_leads last)
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS tenant_settings;
DROP TABLE IF EXISTS tenant_module_leads;

COMMIT;

-- Sprint 4 Welle 1a rollback — drop RLS foundation.
-- Safe to run before any table activates RLS via these helpers. After Welle 2
-- starts dropping these functions while policies still reference them will
-- fail with dependency errors — that is intentional, force the operator to
-- explicitly disable each policy first.

BEGIN;

DROP PROCEDURE IF EXISTS enable_tenant_rls_via_join(text, text, text, text);
DROP PROCEDURE IF EXISTS enable_tenant_rls(text);
DROP FUNCTION IF EXISTS is_system_context();
DROP FUNCTION IF EXISTS current_app_role();
DROP FUNCTION IF EXISTS current_user_id();
DROP FUNCTION IF EXISTS current_tenant_id();

DO $$
BEGIN
    EXECUTE format('ALTER DATABASE %I RESET app.role', current_database());
    EXECUTE format('ALTER DATABASE %I RESET app.user_id', current_database());
    EXECUTE format('ALTER DATABASE %I RESET app.tenant_id', current_database());
END$$;

COMMIT;

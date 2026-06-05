-- Sprint 4 Welle 4.3 — RLS Welle 4c: HR role-based access on hr_employee_documents.
-- 2026-05-14
--
-- Welle 3 (mig 123) activated tenant_isolation on hr_employee_documents — every
-- user inside a tenant could see every other employee's docs. That is wrong:
-- HR documents have a visibility column inherited from the document category
-- that distinguishes hr_only / manager / employee. This migration replaces the
-- simple tenant_isolation policy with a role-aware one that honours visibility:
--
--   hr_only  -> hr_admin / admin role only
--   manager  -> hr_admin + manager role
--   employee -> hr_admin + manager + the employee themself (self-access)
--
-- The role check reads a new session GUC app.user_roles (comma-separated text)
-- set by BeginRLSTx in internal/database/transaction.go.

BEGIN;

-- ============================================================================
-- Helper: comma-separated GUC membership check
-- ============================================================================

CREATE OR REPLACE FUNCTION current_user_has_hr_role(required_role text) RETURNS boolean
LANGUAGE plpgsql STABLE AS $$
DECLARE
    roles_csv text;
BEGIN
    roles_csv := current_setting('app.user_roles', true);
    IF roles_csv IS NULL OR roles_csv = '' THEN
        RETURN false;
    END IF;
    -- Match the role as a whole token, not a substring.
    -- "manager" must not match inside "non-manager" or "manager_assistant".
    RETURN (',' || roles_csv || ',') ILIKE ('%,' || required_role || ',%');
END;
$$;

-- ============================================================================
-- Default GUC at database level so SET LOCAL has something to override
-- ============================================================================

DO $$
BEGIN
    EXECUTE format('ALTER DATABASE %I SET app.user_roles = ''''', current_database());
END$$;

-- ============================================================================
-- Replace tenant_isolation on hr_employee_documents with role-aware policy
-- ============================================================================

DROP POLICY IF EXISTS tenant_isolation ON hr_employee_documents;

CREATE POLICY hr_document_access ON hr_employee_documents
USING (
    tenant_id = current_tenant_id()
    AND (
        is_system_context()
        OR current_user_has_hr_role('admin')
        OR current_user_has_hr_role('hr_admin')
        OR (
            current_user_has_hr_role('manager')
            AND EXISTS (
                SELECT 1 FROM hr_document_categories c
                WHERE c.id = hr_employee_documents.category_id
                  AND c.visibility IN ('manager', 'employee')
            )
        )
        OR (
            employee_id = current_user_id()
            AND EXISTS (
                SELECT 1 FROM hr_document_categories c
                WHERE c.id = hr_employee_documents.category_id
                  AND c.visibility = 'employee'
            )
        )
    )
)
WITH CHECK (
    tenant_id = current_tenant_id()
    AND (
        is_system_context()
        OR current_user_has_hr_role('admin')
        OR current_user_has_hr_role('hr_admin')
    )
);

COMMIT;

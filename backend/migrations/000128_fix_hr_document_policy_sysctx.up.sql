-- Sprint 4 — Fix hr_employee_documents RLS policy: is_system_context() must be
-- checked BEFORE the tenant_id equality test, not nested inside it.
--
-- Bug in 000127: the USING clause was
--   tenant_id = current_tenant_id() AND (is_system_context() OR ...)
-- Under system context current_tenant_id() returns NULL, so
--   tenant_id = NULL  →  always false  →  system context cannot read or insert.
--
-- Fix: hoist is_system_context() to the outer OR, matching the standard pattern
-- from migration 000118:
--   is_system_context() OR (tenant_id = current_tenant_id() AND ...)
--
-- Migration 000127 is already applied in production — this migration replaces
-- the policy without touching the helper function or the GUC default.

BEGIN;

DROP POLICY IF EXISTS hr_document_access ON hr_employee_documents;

CREATE POLICY hr_document_access ON hr_employee_documents
USING (
    is_system_context()
    OR (
        tenant_id = current_tenant_id()
        AND (
            current_user_has_hr_role('admin')
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
)
WITH CHECK (
    is_system_context()
    OR (
        tenant_id = current_tenant_id()
        AND (
            current_user_has_hr_role('admin')
            OR current_user_has_hr_role('hr_admin')
        )
    )
);

COMMIT;

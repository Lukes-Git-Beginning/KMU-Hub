-- Reverse 000128: restore the 000127 policy (is_system_context nested under tenant_id check).
-- Note: this re-introduces the system-context bug intentionally as a rollback target.

BEGIN;

DROP POLICY IF EXISTS hr_document_access ON hr_employee_documents;

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

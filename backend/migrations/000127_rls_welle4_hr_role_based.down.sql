-- Reverse 000127: drop role-aware policy, restore standard tenant_isolation.

BEGIN;

DROP POLICY IF EXISTS hr_document_access ON hr_employee_documents;

CREATE POLICY tenant_isolation ON hr_employee_documents
USING (tenant_id = current_tenant_id() OR is_system_context())
WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

DROP FUNCTION IF EXISTS current_user_has_hr_role(text);

-- Database-level GUC default left in place; it is a no-op for any caller that
-- doesn't read it. Reverting it would require a superuser ALTER DATABASE.

COMMIT;

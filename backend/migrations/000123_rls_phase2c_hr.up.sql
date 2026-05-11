-- Sprint 4 Welle 3.2 — RLS Phase 2c: HR module (8 hr_* tables)
-- 2026-05-11
--
-- Builds on:
--   - 000046 (CREATE TABLE hr_* — all 8 tables have tenant_id NOT NULL)
--   - 000069 (ADD COLUMN tenant_id ON hr_employee_profiles — NOT NULL DEFAULT sentinel)
--   - 000118 (enable_tenant_rls procedure + helper functions)
--   - 000122 (RLS on 95 long-tail NOT NULL tables)
--   - 000124 (RLS on 41 nullable tables after backfill)
--
-- tenant_id status per table:
--   hr_company_settings     — NOT NULL (mig 046)
--   hr_leave_requests       — NOT NULL (mig 046)
--   hr_leave_balances       — NOT NULL (mig 046)
--   hr_work_time_entries    — NOT NULL (mig 046)
--   hr_employee_profiles    — NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000' (mig 069)
--   hr_leave_types          — NOT NULL (mig 046) + System-Seeds with zero-UUID tenant_id
--   hr_document_categories  — NOT NULL (mig 046) + System-Seeds with zero-UUID tenant_id
--   hr_employee_documents   — NOT NULL (mig 046)
--
-- hr_break_entries has NO tenant_id column — it is a child table of
-- hr_work_time_entries (FK on work_time_entry_id ON DELETE CASCADE).
-- RLS is deferred to Welle 4 via via_join policy if needed; the parent
-- hr_work_time_entries is RLS-enabled here, providing effective isolation.
--
-- System-Seed strategy for hr_leave_types + hr_document_categories:
--   mig 046 seeds both tables with tenant_id = '00000000-0000-0000-0000-000000000000'.
--   The application copies these system seeds per-tenant on first access
--   (is_system = TRUE rows are the template). A custom policy permits reading
--   the zero-UUID rows (USING only; WITH CHECK still enforces real tenant_id),
--   matching the pattern used in other seed-carrying tables.
--
-- HR role-based access (employee_id=self, manager visibility, document
-- visibility column) is EXPLICITLY deferred to Welle 4 — this migration
-- closes tenant-isolation only.
--
-- The migration tx runs with row_security off so ALTER TABLE statements are
-- not subject to policies created in the same transaction.

SET LOCAL row_security = off;

BEGIN;

-- ============================================================================
-- 5 tables with standard tenant_isolation policy
-- ============================================================================

CALL enable_tenant_rls('hr_company_settings');
CALL enable_tenant_rls('hr_leave_requests');
CALL enable_tenant_rls('hr_leave_balances');
CALL enable_tenant_rls('hr_work_time_entries');
CALL enable_tenant_rls('hr_employee_documents');

-- ============================================================================
-- hr_employee_profiles — standard policy
-- (tenant_id NOT NULL with default sentinel from mig 069; standard policy
-- is correct: USING (tenant_id = current_tenant_id() OR is_system_context()))
-- ============================================================================

CALL enable_tenant_rls('hr_employee_profiles');

-- ============================================================================
-- hr_leave_types — custom System-Seed-OR-Tenant policy
-- Seeds: 10 rows with tenant_id = '00000000-0000-0000-0000-000000000000'
-- USING: permits seed rows (zero-UUID) + current-tenant rows + system context
-- WITH CHECK: only real tenant rows or system context (guards INSERTs/UPDATEs)
-- ============================================================================

ALTER TABLE hr_leave_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE hr_leave_types FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON hr_leave_types
    USING (
        tenant_id = current_tenant_id()
        OR tenant_id = '00000000-0000-0000-0000-000000000000'::uuid
        OR is_system_context()
    )
    WITH CHECK (
        tenant_id = current_tenant_id()
        OR is_system_context()
    );

-- ============================================================================
-- hr_document_categories — custom System-Seed-OR-Tenant policy
-- Seeds: 4 rows with tenant_id = '00000000-0000-0000-0000-000000000000'
-- Same USING/WITH CHECK pattern as hr_leave_types above.
-- ============================================================================

ALTER TABLE hr_document_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE hr_document_categories FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON hr_document_categories
    USING (
        tenant_id = current_tenant_id()
        OR tenant_id = '00000000-0000-0000-0000-000000000000'::uuid
        OR is_system_context()
    )
    WITH CHECK (
        tenant_id = current_tenant_id()
        OR is_system_context()
    );

COMMIT;

-- Sprint 4 Welle 1a — Row-Level-Security foundation.
-- 2026-05-10
--
-- This migration is foundational: it creates the helper functions and
-- procedures that subsequent migrations (000120, 000121) will use to
-- enable per-tenant RLS policies on individual tables. No tables are
-- altered here; no policies are activated.
--
-- Design:
-- * Three session-level GUCs (app.tenant_id, app.user_id, app.role) are
--   declared at the database level so any pool connection can SET them
--   inside a transaction.
-- * current_tenant_id() / current_user_id() / current_app_role() are
--   STABLE plpgsql wrappers around current_setting(..., true) that
--   gracefully return NULL/'' when the setting is missing or empty.
-- * is_system_context() returns true when the connection is operating
--   in system-bypass mode (workers, migrations, audit-inserts).
-- * enable_tenant_rls(table_name) is the standard activation procedure:
--   ALTER TABLE ... ENABLE + FORCE + CREATE POLICY tenant_isolation
--   USING (tenant_id = current_tenant_id() OR is_system_context()).
-- * enable_tenant_rls_via_join(...) is a fallback for child tables
--   that derive tenant from a parent FK (rare — own column preferred).

BEGIN;

-- 1) Database-level defaults so SET LOCAL inside transactions sees an
--    initialized GUC. Empty string is the "no tenant" sentinel — treated
--    by current_tenant_id() as NULL.
ALTER DATABASE kmuhub SET app.tenant_id = '';
ALTER DATABASE kmuhub SET app.user_id = '';
ALTER DATABASE kmuhub SET app.role = '';

-- 2) Helper functions. STABLE so the planner can cache them within a
--    statement but re-evaluate per query (cannot be IMMUTABLE — they
--    depend on session state).

CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS uuid
LANGUAGE plpgsql STABLE AS $$
DECLARE
    v text;
BEGIN
    v := current_setting('app.tenant_id', true);
    IF v IS NULL OR v = '' THEN
        RETURN NULL;
    END IF;
    RETURN v::uuid;
EXCEPTION WHEN invalid_text_representation THEN
    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION current_user_id() RETURNS uuid
LANGUAGE plpgsql STABLE AS $$
DECLARE
    v text;
BEGIN
    v := current_setting('app.user_id', true);
    IF v IS NULL OR v = '' THEN
        RETURN NULL;
    END IF;
    RETURN v::uuid;
EXCEPTION WHEN invalid_text_representation THEN
    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION current_app_role() RETURNS text
LANGUAGE plpgsql STABLE AS $$
BEGIN
    RETURN COALESCE(current_setting('app.role', true), '');
END;
$$;

CREATE OR REPLACE FUNCTION is_system_context() RETURNS boolean
LANGUAGE plpgsql STABLE AS $$
BEGIN
    RETURN current_app_role() = 'system';
END;
$$;

-- 3) Standard activation procedure — invoked once per table from later
--    migrations. FORCE ROW LEVEL SECURITY ensures the policy applies even
--    to the table owner (Postgres default exempts the owner).
CREATE OR REPLACE PROCEDURE enable_tenant_rls(target_table text)
LANGUAGE plpgsql AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', target_table);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', target_table);
    EXECUTE format(
        'CREATE POLICY tenant_isolation ON %I '
        'USING (tenant_id = current_tenant_id() OR is_system_context()) '
        'WITH CHECK (tenant_id = current_tenant_id() OR is_system_context())',
        target_table
    );
END;
$$;

-- 4) Fallback for child tables that derive tenant via JOIN instead of
--    carrying their own column. Welle 1b adds explicit columns to all
--    Pilot-0 child tables, so this is documented but unused initially.
CREATE OR REPLACE PROCEDURE enable_tenant_rls_via_join(
    child_table text,
    parent_table text,
    fk_column text,
    parent_pk text DEFAULT 'id'
)
LANGUAGE plpgsql AS $$
BEGIN
    EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', child_table);
    EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', child_table);
    EXECUTE format(
        'CREATE POLICY tenant_isolation ON %I '
        'USING (EXISTS (SELECT 1 FROM %I p WHERE p.%I = %I.%I AND (p.tenant_id = current_tenant_id() OR is_system_context()))) '
        'WITH CHECK (EXISTS (SELECT 1 FROM %I p WHERE p.%I = %I.%I AND (p.tenant_id = current_tenant_id() OR is_system_context())))',
        child_table,
        parent_table, parent_pk, child_table, fk_column,
        parent_table, parent_pk, child_table, fk_column
    );
END;
$$;

COMMIT;

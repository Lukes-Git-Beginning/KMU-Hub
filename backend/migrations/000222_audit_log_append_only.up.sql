-- ============================================================================
-- audit_log: DB-level append-only enforcement (GoBD / §257 / §147 AO).
--
-- Purpose: Prohibit UPDATE and DELETE on audit_log rows at the database level
-- so that no application bug, misconfigured role, or direct DB access can
-- silently alter the immutable audit trail required under German commercial and
-- tax record-retention law (§257 HGB / §147 AO / GoBD).
--
-- Scope: Row-level BEFORE trigger fires for every UPDATE and DELETE attempt
-- and raises an exception, preventing the write.
--
-- Not affected:
--   • INSERT (append — this is the only permitted write).
--   • Partition management DDL (e.g. ATTACH / DETACH / DROP PARTITION TABLE)
--     that operates outside of DML row triggers; retention DROP of whole
--     partition tables remains a DDL-level operation and is unaffected.
--   • Rows deleted via CleanupRow in RLS integration tests use system-context;
--     test cleanup (DELETE via testutil.CleanupRow) calls defer cleanup which
--     also fires the trigger — tests must use testutil.CleanupRow only on
--     mutable test fixtures, NOT on audit_log rows in prod-like environments.
-- ============================================================================

CREATE OR REPLACE FUNCTION prevent_audit_log_mutation() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only (GoBD/§257/§147 AO)';
END;
$$;

CREATE TRIGGER audit_log_append_only
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_mutation();

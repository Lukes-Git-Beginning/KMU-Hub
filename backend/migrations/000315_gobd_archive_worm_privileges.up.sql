-- ============================================================================
-- GoBD-Belegarchiv: enforce immutability in the database, not just in Go.
--
-- 000139 created the archive as an append-only store (§ 147 AO,
-- "Unveraenderbarkeit"), but nothing below the application enforced that.
-- 000121_create_app_role.up.sql:49 grants kmuhub_app a schema-wide
--   GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public
-- plus matching ALTER DEFAULT PRIVILEGES, so both archive tables carry UPDATE
-- and DELETE for the role every service runs as. Any future query, any handler
-- and any SQL injection could rewrite or erase an archived document.
--
-- Safe to revoke: the Go code never issues UPDATE or DELETE against either
-- table. The Repository interface (internal/biz/gobdarchive/repository.go)
-- exposes exactly Create, GetByID, List, AppendEvent, ListEvents; a grep over
-- internal/ for "UPDATE gobd" and "DELETE FROM gobd" returns zero non-test
-- hits (verified 2026-08-22).
--
-- Per table:
--   gobd_documents        The archived document itself. Nothing may ever
--                         change it; the hash column exists so tampering is
--                         detectable, and this makes it impossible through
--                         the app role in the first place.
--   gobd_document_events  The access/annotation trail of those documents. A
--                         trail that the app can delete is not a trail. Note
--                         that this is NOT a plain join table -- it carries
--                         its own payload (event_type, note, metadata) and
--                         has no legitimate delete path.
--
-- Deliberately NOT revoked:
--   - Nothing is revoked from kmuhub. Migrations run under that role and must
--     keep being able to change the archive structurally.
--   - The referential actions on both tables (source_invoice_id ON DELETE SET
--     NULL, tenant_id ON DELETE CASCADE) keep working: PostgreSQL executes RI
--     triggers as the table owner, not as the calling role.
--
-- Future gobd_* tables: ALTER DEFAULT PRIVILEGES in 000121 section 4 will hand
-- kmuhub_app UPDATE and DELETE on them automatically. Any migration adding a
-- table to this archive must repeat the REVOKE below.
-- TestGobdArchive_WormPrivileges_AllArchiveTables in
-- internal/biz/gobdarchive/worm_privileges_db_test.go scans every public
-- gobd_% table and fails if one of them carries UPDATE or DELETE, so the
-- omission surfaces as a red test rather than as a silently writable archive.
-- ============================================================================

BEGIN;

REVOKE UPDATE, DELETE ON gobd_documents       FROM kmuhub_app;
REVOKE UPDATE, DELETE ON gobd_document_events FROM kmuhub_app;

COMMIT;

-- Close a real tenant gap flagged in migration 000253 (unit wp-chat): projects
-- carries a NOT NULL tenant_id (since mig 000106, same tenant_id_retrofit
-- batch as channels/messages) but never received an RLS policy. Every read
-- and write path in internal/work/project/postgres_repository.go already
-- scopes explicitly on tenant_id (verified before writing this migration),
-- so this closes the database-level backstop, not a live data leak.

SET LOCAL row_security = off;

BEGIN;

CALL enable_tenant_rls('projects');

COMMIT;

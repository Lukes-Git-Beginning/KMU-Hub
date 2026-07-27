-- Close a real tenant gap found while adding DB-backed write tests for
-- internal/chat (unit wp-chat): channels and messages both carry a NOT NULL
-- tenant_id (since mig 000106) but never received an RLS policy. Every
-- sibling table in the same Option-B batch (channel_memberships,
-- message_mentions, chat_files, guest_sessions, ...) was RLS-enabled in
-- mig 000122 — channels and messages were dropped from that sweep. The
-- gateway/service layer scopes reads and writes correctly (GetByIDForTenant,
-- explicit tenant_id predicates), but without RLS a raw kmuhub_app query or a
-- future missed WHERE-clause has no database-level backstop, on the two most
-- central chat tables in the schema.
--
-- projects (also in the mig 000106 tenant_id_retrofit list) has the same gap
-- but belongs to a different module — out of scope here, flagged in the loop
-- journal for a follow-up unit.

SET LOCAL row_security = off;

BEGIN;

CALL enable_tenant_rls('channels');
CALL enable_tenant_rls('messages');

COMMIT;

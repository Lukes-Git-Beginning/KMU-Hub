-- Reverse of 000253 — drop the tenant_isolation policy and disable RLS on
-- channels and messages.

SET LOCAL row_security = off;

BEGIN;

DROP POLICY IF EXISTS tenant_isolation ON messages;
ALTER TABLE messages DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON channels;
ALTER TABLE channels DISABLE ROW LEVEL SECURITY;

COMMIT;

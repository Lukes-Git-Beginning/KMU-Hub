-- Reverse of 000254 — drop the tenant_isolation policy and disable RLS on
-- projects.

SET LOCAL row_security = off;

BEGIN;

DROP POLICY IF EXISTS tenant_isolation ON projects;
ALTER TABLE projects DISABLE ROW LEVEL SECURITY;

COMMIT;

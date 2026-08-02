-- Reverse of 000271. The backfill is not reversible — rows deleted in step 3
-- were unattributable and stay gone — but the schema returns to its 000242 shape.

DROP POLICY IF EXISTS tenant_isolation ON events;
ALTER TABLE events NO FORCE ROW LEVEL SECURITY;
ALTER TABLE events DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_events_tenant_id;

ALTER TABLE events DROP COLUMN IF EXISTS tenant_id;

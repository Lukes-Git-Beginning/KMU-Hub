-- Reverse of 000275. Collapsing per-tenant usage back into one global row is
-- lossy in a different way than 000273/000274: there is no single "most
-- recent" reading to prefer. Sum every tenant's used_bytes back together to
-- preserve the true aggregate byte count, and keep the most recently edited
-- max_bytes as the surviving quota policy.

DROP POLICY IF EXISTS tenant_isolation ON storage_quotas;
ALTER TABLE storage_quotas NO FORCE ROW LEVEL SECURITY;
ALTER TABLE storage_quotas DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_storage_quotas_tenant;

WITH collapsed AS (
    SELECT COALESCE(SUM(used_bytes), 0) AS used_bytes FROM storage_quotas
),
latest AS (
    SELECT id, max_bytes FROM storage_quotas ORDER BY updated_at DESC, id LIMIT 1
)
UPDATE storage_quotas
   SET max_bytes = latest.max_bytes,
       used_bytes = collapsed.used_bytes,
       updated_at = NOW()
  FROM latest, collapsed
 WHERE storage_quotas.id = latest.id;

DELETE FROM storage_quotas
 WHERE id NOT IN (SELECT id FROM storage_quotas ORDER BY updated_at DESC, id LIMIT 1);

ALTER TABLE storage_quotas DROP COLUMN IF EXISTS tenant_id;

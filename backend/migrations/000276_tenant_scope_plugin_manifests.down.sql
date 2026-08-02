-- Reverse of 000276. Lossy in one direction the previous three were not:
-- restoring a globally unique slug means that where two tenants defined a
-- manifest under the same slug, only one row can survive, and dropping the
-- others cascades into their plugin_installations. The catalogue row
-- (tenant_id IS NULL) wins, then the oldest — the same preference
-- ManifestRepository.GetBySlug applies while the column exists.

DROP POLICY IF EXISTS tenant_isolation_read ON plugin_manifests;
DROP POLICY IF EXISTS tenant_isolation_write ON plugin_manifests;
ALTER TABLE plugin_manifests NO FORCE ROW LEVEL SECURITY;
ALTER TABLE plugin_manifests DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_plugin_manifests_tenant_slug;

DELETE FROM plugin_manifests
 WHERE id NOT IN (
     SELECT DISTINCT ON (slug) id
       FROM plugin_manifests
      ORDER BY slug, (tenant_id IS NULL) DESC, created_at, id
 );

ALTER TABLE plugin_manifests ADD CONSTRAINT plugin_manifests_slug_key UNIQUE (slug);

ALTER TABLE plugin_manifests DROP COLUMN IF EXISTS tenant_id;

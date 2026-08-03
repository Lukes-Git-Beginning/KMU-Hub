-- ============================================================================
-- Last of the five tables found by backlog unit g-rls-tenant-scoped-admin-writes,
-- after 000273 (two_factor_policy), 000274 (presence_config, dashboard_defaults)
-- and 000275 (storage_quotas).
--
-- plugin_manifests had no tenant_id at all: POST /api/v1/plugins/manifests is
-- guarded by RequireRole("admin") (route_plugin.go:53), which checks "is an
-- admin" but never "an admin of which tenant", and GET /manifests reads the
-- table without a filter. Every config manifest a tenant defines is therefore
-- visible to — and installable by — every other tenant on the installation.
-- (WASM manifests are separately refused while plugins.wasm is off; config
-- manifests are not.)
--
-- The same gap makes DELETE /manifests/{id} destructive across tenants:
-- Service.DeleteManifest refuses to delete a manifest that still has active
-- installations, but it counts them through plugin_installations, which HAS
-- had RLS since 000122. A foreign tenant's installations are invisible to the
-- counting query, so the count comes back 0, the delete proceeds, and
-- plugin_installations.manifest_id ON DELETE CASCADE removes that tenant's
-- installation with it.
--
-- SCHEMA DECISION — nullable tenant_id, the roles/role_permissions pattern of
-- 000256, chosen over the two alternatives the backlog unit named:
--   * "tenant operation" (tenant_id NOT NULL) cannot express a catalogue of
--     manifests shipped by Zentria for everyone, which is what the existing
--     rows are and what a plugin marketplace needs.
--   * "platform operation" (a platform_admin-only route) would take the
--     ability to define a config plugin away from tenants entirely and needs a
--     role that no HTTP route grants today.
-- NULL means "shipped with the product, readable by every tenant, writable by
-- none"; a non-NULL tenant_id means "this tenant's own manifest". Reads see
-- both, writes only their own — exactly the asymmetry roles/role_permissions
-- have carried since 000256, so the schema keeps one pattern rather than two.
-- Catalogue rows are seeded by migration, the same way the role presets are;
-- no HTTP route creates them.
--
-- No backfill: existing rows keep tenant_id NULL. That is the lossless reading
-- of the old data — those manifests WERE visible to every tenant, and they
-- stay visible to every tenant. What changes is that a tenant admin can no
-- longer delete or overwrite them, which is the cross-tenant deletion above.
-- ============================================================================

ALTER TABLE plugin_manifests
    ADD COLUMN tenant_id UUID NULL REFERENCES tenants(id) ON DELETE CASCADE;

-- Slugs are unique per tenant, not globally: two tenants may both define a
-- manifest called "quote-approval". COALESCE maps the catalogue rows onto a
-- fixed sentinel so they keep colliding with each other, byte-identical to the
-- expression 000256 uses for roles.
ALTER TABLE plugin_manifests DROP CONSTRAINT plugin_manifests_slug_key;
CREATE UNIQUE INDEX idx_plugin_manifests_tenant_slug
    ON plugin_manifests (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid), slug);

-- idx_plugin_manifests_slug stays: the expression index above leads with the
-- tenant, so it cannot serve GetBySlug's plain slug lookup.

-- RLS. Not enable_tenant_rls(): that procedure writes the symmetric policy
-- (tenant_id = current_tenant_id()), under which no tenant would see the
-- catalogue rows at all. Same asymmetric pair as roles.
ALTER TABLE plugin_manifests ENABLE ROW LEVEL SECURITY;
ALTER TABLE plugin_manifests FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_read ON plugin_manifests
    FOR SELECT
    USING (tenant_id IS NULL OR tenant_id = current_tenant_id() OR is_system_context());

CREATE POLICY tenant_isolation_write ON plugin_manifests
    FOR ALL
    USING (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

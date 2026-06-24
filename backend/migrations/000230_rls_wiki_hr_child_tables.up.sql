-- Welle 5 — RLS for wiki and HR child tables that lacked tenant_id.
-- 2026-06-24
--
-- Deferred tables (previously noted in mig 000122 and 000123):
--   wiki_versions        — FK article_id → wiki_articles.tenant_id
--   wiki_attachments     — FK article_id → wiki_articles.tenant_id
--   wiki_share_tokens    — FK article_id → wiki_articles.tenant_id
--   hr_break_entries     — FK work_time_entry_id → hr_work_time_entries.tenant_id
--
-- All four tables have no tenant_id column of their own; we ADD COLUMN,
-- JOIN-backfill from the RLS-enabled parent, then call enable_tenant_rls()
-- for a direct tenant_id check (faster than via_join at query time).
-- Orphaned rows (parent deleted without CASCADE — should not exist given
-- ON DELETE CASCADE on all FKs) receive the zero-UUID sentinel.
--
-- Requires:
--   000118 (enable_tenant_rls + helper functions)
--   000122 (wiki_articles and wiki_categories RLS active)
--   000123 (hr_work_time_entries RLS active)

SET LOCAL row_security = off;

BEGIN;

-- ============================================================================
-- wiki_versions (mig 076) — FK article_id → wiki_articles.tenant_id
-- ============================================================================

ALTER TABLE wiki_versions ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE wiki_versions wv
SET tenant_id = wa.tenant_id
FROM wiki_articles wa
WHERE wv.article_id = wa.id
  AND wv.tenant_id IS NULL;
UPDATE wiki_versions SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE wiki_versions ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE wiki_versions
    ADD CONSTRAINT fk_wiki_versions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_wiki_versions_tenant_id ON wiki_versions(tenant_id);
CALL enable_tenant_rls('wiki_versions');

-- ============================================================================
-- wiki_attachments (mig 076) — FK article_id → wiki_articles.tenant_id
-- ============================================================================

ALTER TABLE wiki_attachments ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE wiki_attachments watt
SET tenant_id = wa.tenant_id
FROM wiki_articles wa
WHERE watt.article_id = wa.id
  AND watt.tenant_id IS NULL;
UPDATE wiki_attachments SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE wiki_attachments ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE wiki_attachments
    ADD CONSTRAINT fk_wiki_attachments_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_wiki_attachments_tenant_id ON wiki_attachments(tenant_id);
CALL enable_tenant_rls('wiki_attachments');

-- ============================================================================
-- wiki_share_tokens (mig 076) — FK article_id → wiki_articles.tenant_id
-- ============================================================================

ALTER TABLE wiki_share_tokens ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE wiki_share_tokens wst
SET tenant_id = wa.tenant_id
FROM wiki_articles wa
WHERE wst.article_id = wa.id
  AND wst.tenant_id IS NULL;
UPDATE wiki_share_tokens SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE wiki_share_tokens ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE wiki_share_tokens
    ADD CONSTRAINT fk_wiki_share_tokens_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_wiki_share_tokens_tenant_id ON wiki_share_tokens(tenant_id);
CALL enable_tenant_rls('wiki_share_tokens');

-- ============================================================================
-- hr_break_entries (mig 046) — FK work_time_entry_id → hr_work_time_entries.tenant_id
-- ============================================================================

ALTER TABLE hr_break_entries ADD COLUMN IF NOT EXISTS tenant_id UUID;
UPDATE hr_break_entries hbe
SET tenant_id = hwte.tenant_id
FROM hr_work_time_entries hwte
WHERE hbe.work_time_entry_id = hwte.id
  AND hbe.tenant_id IS NULL;
UPDATE hr_break_entries SET tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
WHERE tenant_id IS NULL;
ALTER TABLE hr_break_entries ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE hr_break_entries
    ADD CONSTRAINT fk_hr_break_entries_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
    NOT VALID;
CREATE INDEX IF NOT EXISTS idx_hr_break_entries_tenant_id ON hr_break_entries(tenant_id);
CALL enable_tenant_rls('hr_break_entries');

COMMIT;

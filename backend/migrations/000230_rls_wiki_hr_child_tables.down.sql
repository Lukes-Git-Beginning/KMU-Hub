-- Rollback Welle 5 — remove RLS policies and tenant_id columns
-- from wiki_versions, wiki_attachments, wiki_share_tokens, hr_break_entries.

BEGIN;

-- ============================================================================
-- wiki_versions
-- ============================================================================
DROP POLICY IF EXISTS tenant_isolation ON wiki_versions;
ALTER TABLE wiki_versions DISABLE ROW LEVEL SECURITY;
ALTER TABLE wiki_versions DROP CONSTRAINT IF EXISTS fk_wiki_versions_tenant;
DROP INDEX IF EXISTS idx_wiki_versions_tenant_id;
ALTER TABLE wiki_versions DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- wiki_attachments
-- ============================================================================
DROP POLICY IF EXISTS tenant_isolation ON wiki_attachments;
ALTER TABLE wiki_attachments DISABLE ROW LEVEL SECURITY;
ALTER TABLE wiki_attachments DROP CONSTRAINT IF EXISTS fk_wiki_attachments_tenant;
DROP INDEX IF EXISTS idx_wiki_attachments_tenant_id;
ALTER TABLE wiki_attachments DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- wiki_share_tokens
-- ============================================================================
DROP POLICY IF EXISTS tenant_isolation ON wiki_share_tokens;
ALTER TABLE wiki_share_tokens DISABLE ROW LEVEL SECURITY;
ALTER TABLE wiki_share_tokens DROP CONSTRAINT IF EXISTS fk_wiki_share_tokens_tenant;
DROP INDEX IF EXISTS idx_wiki_share_tokens_tenant_id;
ALTER TABLE wiki_share_tokens DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- hr_break_entries
-- ============================================================================
DROP POLICY IF EXISTS tenant_isolation ON hr_break_entries;
ALTER TABLE hr_break_entries DISABLE ROW LEVEL SECURITY;
ALTER TABLE hr_break_entries DROP CONSTRAINT IF EXISTS fk_hr_break_entries_tenant;
DROP INDEX IF EXISTS idx_hr_break_entries_tenant_id;
ALTER TABLE hr_break_entries DROP COLUMN IF EXISTS tenant_id;

COMMIT;

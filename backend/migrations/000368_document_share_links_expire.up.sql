-- ============================================================================
-- Migration 000368: every document share link expires, at the latest one year
-- after it was minted.
--
-- Decision 2026-10-25 (see 000352 for wiki, form and report links): no share
-- link without an end, at most 365 days. document_share_links (000266) kept
-- expires_at NULL = never expires. The service now defaults a missing expiry to
-- now + 365 days and refuses anything further out.
--
-- The backfill gives existing rows the end they would have got at creation:
-- created_at + 365 days. A link older than a year therefore expires with this
-- migration -- that is the point of the decision, not a side effect. Rows that
-- already carry an expiry keep it.
--
-- No RLS change: the tenant_isolation policy is column-independent.
-- ============================================================================

BEGIN;

UPDATE document_share_links SET expires_at = created_at + INTERVAL '365 days' WHERE expires_at IS NULL;
ALTER TABLE document_share_links ALTER COLUMN expires_at SET NOT NULL;

COMMIT;

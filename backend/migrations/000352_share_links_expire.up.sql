-- ============================================================================
-- Migration 000352: every share link expires, at the latest one year after
-- it was minted.
--
-- Decision 2026-10-25: no share link without an end, at most 365 days. Wiki
-- (000076) and form (000293) links had no expiry unless the author set one;
-- report links (000252) the same, capped at 365 days when set. The services
-- now default a missing expiry to now + 365 days (internal/linksecret.Expiry,
-- berichte maxShareExpiryDays) and refuse anything further out.
--
-- The backfill gives the existing rows the end they would have got at
-- creation: created_at + 365 days. A link older than a year therefore expires
-- with this migration -- that is the point of the decision, not a side effect.
-- Rows that already carry an expiry keep it, including ones further out than a
-- year: they were minted under the old rule and cut by revoke, not here.
--
-- NOT NULL afterwards makes "no link without an end" a property of the table
-- rather than of the three services that write it.
--
-- No RLS change: the tenant_isolation policies are column-independent.
-- ============================================================================

BEGIN;

UPDATE wiki_share_tokens SET expires_at = created_at + INTERVAL '365 days' WHERE expires_at IS NULL;
ALTER TABLE wiki_share_tokens ALTER COLUMN expires_at SET NOT NULL;

UPDATE form_share_tokens SET expires_at = created_at + INTERVAL '365 days' WHERE expires_at IS NULL;
ALTER TABLE form_share_tokens ALTER COLUMN expires_at SET NOT NULL;

UPDATE report_share_tokens SET expires_at = created_at + INTERVAL '365 days' WHERE expires_at IS NULL;
ALTER TABLE report_share_tokens ALTER COLUMN expires_at SET NOT NULL;

COMMIT;

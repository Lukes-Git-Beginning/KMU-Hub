-- ============================================================================
-- Migration 000367: store report share tokens only as SHA-256.
--
-- report_share_tokens (000252) kept the link secret in clear text, the last
-- share table to do so after wiki and forms (000351). Whoever reads the table
-- -- a backup, a wrong join -- held working read links to tenant reports.
-- Decision 2026-10-25, variant a on 2026-10-26: hash only (internal/linksecret),
-- after the desktop copied the link from the create response alone
-- (fe-berichte-share-link-shown-once, 4e5c830d).
--
-- Existing links keep working: the backfill hashes the stored token the same
-- way linksecret.Hash does (hex SHA-256 of the UTF-8 bytes; sha256() is core
-- since PG 11, no pgcrypto), and the lookup hashes what the visitor presents.
-- Dropping `token` drops its UNIQUE constraint with it; the new UNIQUE on
-- token_hash is the lookup index and keeps the cross-tenant collision guard.
--
-- No RLS change: the tenant_isolation policy is column-independent.
-- ============================================================================

BEGIN;

ALTER TABLE report_share_tokens ADD COLUMN token_hash TEXT;
UPDATE report_share_tokens SET token_hash = encode(sha256(convert_to(token, 'UTF8')), 'hex');
ALTER TABLE report_share_tokens
    ALTER COLUMN token_hash SET NOT NULL,
    ADD CONSTRAINT report_share_tokens_token_hash_key UNIQUE (token_hash),
    DROP COLUMN token;

COMMIT;

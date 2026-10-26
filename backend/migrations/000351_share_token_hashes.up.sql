-- ============================================================================
-- Migration 000351: store wiki and form share tokens only as SHA-256.
--
-- wiki_share_tokens (000076) and form_share_tokens (000293) kept the link
-- secret in clear text. Whoever reads the table -- a backup, a wrong join --
-- held working external links: read access to a wiki page, write access to a
-- form. Decision 2026-10-25: hash only (internal/linksecret).
--
-- 000293 argued for clear text so the author could copy the link again later.
-- No client does that: the wiki share dialog never reads the token, and the
-- form panel copies `url`, which the gateway sends empty until the public
-- fill-out page exists. The plain token now leaves the server once, in the
-- create response.
--
-- Existing links keep working: the backfill hashes the stored token the same
-- way linksecret.Hash does (hex SHA-256 of the UTF-8 bytes; sha256() is core
-- since PG 11, no pgcrypto), and the lookup hashes what the visitor presents.
-- Dropping `token` drops its UNIQUE constraint and idx_wiki_share_tokens_token
-- with it; the new UNIQUE on token_hash is the lookup index and keeps the
-- cross-tenant collision guard 000293 put on the clear column.
--
-- report_share_tokens (000252) is not touched: its trade is argued separately
-- there.
--
-- No RLS change: both tenant_isolation policies are column-independent.
-- ============================================================================

BEGIN;

ALTER TABLE wiki_share_tokens ADD COLUMN token_hash TEXT;
UPDATE wiki_share_tokens SET token_hash = encode(sha256(convert_to(token, 'UTF8')), 'hex');
ALTER TABLE wiki_share_tokens
    ALTER COLUMN token_hash SET NOT NULL,
    ADD CONSTRAINT wiki_share_tokens_token_hash_key UNIQUE (token_hash),
    DROP COLUMN token;

ALTER TABLE form_share_tokens ADD COLUMN token_hash TEXT;
UPDATE form_share_tokens SET token_hash = encode(sha256(convert_to(token, 'UTF8')), 'hex');
ALTER TABLE form_share_tokens
    ALTER COLUMN token_hash SET NOT NULL,
    ADD CONSTRAINT form_share_tokens_token_hash_key UNIQUE (token_hash),
    DROP COLUMN token;

COMMIT;

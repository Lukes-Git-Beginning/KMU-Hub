-- ============================================================================
-- Migration 000297: soft revocation and authorship for wiki_share_tokens.
--
-- These tokens exist since 000076 but were inert until loop run 5 built
-- POST /api/v1/public/wiki/articles/{token}. That turned them into a
-- credential redeemable from outside, and the table records neither who
-- handed one out nor that one was ever cut.
--
-- Revocation is soft, same trade as form_share_tokens (000293): a hard DELETE
-- answers the same 404, but "this link was cut" and "this link never existed"
-- are different facts for an author looking at an external READ path on their
-- own knowledge base. When the question later is "which articles were
-- readable from outside, on whose authority, and until when", a deleted row
-- has no answer. expires_at stays nullable -- a link without an end date is a
-- deliberate choice -- so revoked_at is the only way to end one early.
--
-- created_by is nullable and stays nullable: every token minted before this
-- migration has no known author, and ON DELETE SET NULL keeps the token
-- (and the audit trail of the cut) intact when the user record goes away.
--
-- No RLS change: policy tenant_isolation from the 000076 retrofit is
-- column-independent and already covers both new columns.
-- ============================================================================

BEGIN;

ALTER TABLE wiki_share_tokens
    ADD COLUMN revoked_at TIMESTAMPTZ NULL,
    ADD COLUMN created_by UUID NULL REFERENCES users (id) ON DELETE SET NULL;

-- The public redemption resolves a token by its secret and then checks
-- usability; the listing reads one article's links newest first. Neither
-- benefits from an index on revoked_at -- it is a per-row predicate on a
-- handful of rows, not a selector.

COMMIT;

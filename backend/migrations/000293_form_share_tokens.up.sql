-- ============================================================================
-- Migration 000293: form_share_tokens (BACKLOG B8 "intake-public-submit").
--
-- Public, unauthenticated form fill-out. `form_schemas.is_public` has been
-- stored since 000081 but was never evaluated as an auth gate anywhere, and
-- the whole /api/v1/formulare group sits behind RequireAuthenticated -- so a
-- "public" form could not actually be filled out by anyone without a login.
--
-- The gate is the TOKEN, not the flag. `is_public` is the precondition for
-- minting one (an author must deliberately open the form up); the token is
-- what a visitor presents. That split matters: flipping is_public back to
-- false must not silently leave a live link behind, so the public lookup
-- re-checks the flag on the schema it resolves to.
--
-- Token in clear text, deliberately, same trade as report_share_tokens
-- (000252): the token is the copyable link the author hands out, and a
-- one-way hash would make "Link kopieren" impossible. It grants the ability
-- to APPEND one form submission to exactly one schema -- it reads nothing.
--
-- Revocation is soft. A hard DELETE would answer 404 just as well, but "this
-- link was cut" and "this link never existed" are different facts for an
-- author looking at an external write path, and submission_count is the only
-- record of how often the link was used before it was cut.
-- ============================================================================

BEGIN;

CREATE TABLE form_share_tokens (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    form_schema_id   UUID        NOT NULL REFERENCES form_schemas (id) ON DELETE CASCADE,
    -- 32 bytes of crypto/rand, base64url-encoded (43 chars). Unique across
    -- ALL tenants: the public route resolves the tenant FROM this column, so
    -- a collision would be a cross-tenant write.
    token            TEXT        NOT NULL UNIQUE,
    -- NULL = never expires.
    expires_at       TIMESTAMPTZ NULL,
    revoked_at       TIMESTAMPTZ NULL,
    -- NULL = unlimited. A spam brake the author sets per link; the strict
    -- per-IP public rate limiter is the brake they do not have to set.
    max_submissions  INTEGER     NULL CHECK (max_submissions IS NULL OR max_submissions > 0),
    submission_count INTEGER     NOT NULL DEFAULT 0,
    created_by       UUID        NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Listing the links of one schema, newest first (the only authenticated read).
CREATE INDEX idx_form_share_tokens_schema
    ON form_share_tokens (tenant_id, form_schema_id, created_at DESC);

-- RLS: standard tenant isolation (matches form_schemas). The public lookup by
-- token runs in the system context because the request carries no tenant yet
-- -- resolving it is the very question the token answers. Everything after
-- that lookup, including the submission INSERT, runs under the resolved
-- tenant and is filtered normally.
ALTER TABLE form_share_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE form_share_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON form_share_tokens
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;

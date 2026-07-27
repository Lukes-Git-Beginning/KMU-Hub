-- Migration 000252: report_share_tokens
-- External, unauthenticated read links for a single report document.
-- berichte-client.ts already calls /api/v1/berichte/documents/:id/shares and
-- /api/v1/berichte/shares/:id; the public read path they hand out did not exist
-- server-side at all.
--
-- The token is stored in clear text, deliberately, and not as a hash:
-- ReportShareToken.token is part of the list response the frontend renders as a
-- copyable link, so a one-way hash would make the existing "Link kopieren"
-- affordance impossible. The security trade is empty here -- a token grants
-- read access to exactly one report_documents row, and anyone holding the dump
-- that leaks the token holds that row too. The password is different: it is
-- user-chosen material that may be reused elsewhere, so it is bcrypt-hashed
-- and never stored or returned in clear.
--
-- Revocation is a soft revoke. A hard DELETE would answer 404 just as well,
-- but "this link was cut" and "this link never existed" are different facts for
-- an admin looking at an external access path, and view_count is the only
-- record of how often the link was used before it was cut.

BEGIN;

CREATE TABLE report_share_tokens (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    document_id   UUID        NOT NULL REFERENCES report_documents (id) ON DELETE CASCADE,
    -- 32 bytes of crypto/rand, base64url-encoded (43 chars). Unique across all
    -- tenants: the public route resolves the tenant FROM this column, so a
    -- collision would be a cross-tenant read.
    token         TEXT        NOT NULL UNIQUE,
    -- NULL = no password. bcrypt hash otherwise, never the password itself.
    password_hash TEXT        NULL,
    -- NULL = never expires.
    expires_at    TIMESTAMPTZ NULL,
    revoked_at    TIMESTAMPTZ NULL,
    view_count    INTEGER     NOT NULL DEFAULT 0,
    created_by    UUID        NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Listing the links of one document, newest first (the only authenticated read).
CREATE INDEX idx_report_share_tokens_document
    ON report_share_tokens (tenant_id, document_id, created_at DESC);

-- RLS: standard tenant isolation (matches report_documents in 000245).
-- The public lookup by token runs in the system context because the request
-- carries no tenant yet -- resolving it is the very question. Everything after
-- the lookup runs under the resolved tenant and is filtered normally.
ALTER TABLE report_share_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_share_tokens FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON report_share_tokens
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;

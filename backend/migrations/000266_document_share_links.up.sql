-- ============================================================================
-- document_share_links: external, unauthenticated read/download links for a
-- single document file (backend-gaps.md §dokumente "Freigabelinks").
-- Optional bcrypt password, optional expiry. Revocation is a soft revoke
-- (revoked_at), mirroring report_share_tokens (000252): "this link was cut"
-- and "this link never existed" stay distinguishable to the tenant admin
-- listing their own links, even though the public redemption route answers
-- both cases identically (see internal/document/file/sharelink.go).
--
-- token is stored in clear text, deliberately, not hashed: it is returned on
-- create/list as the copyable link, same trade-off as report_share_tokens.
-- password_hash is bcrypt, never the password itself.
-- ============================================================================

CREATE TABLE document_share_links (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    file_id       UUID        NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    -- 32 bytes of crypto/rand, base64url-encoded. Unique across all tenants:
    -- the public route resolves the tenant FROM this column.
    token         TEXT        NOT NULL UNIQUE,
    -- NULL = no password. bcrypt hash otherwise, never the password itself.
    password_hash TEXT        NULL,
    -- NULL = never expires.
    expires_at    TIMESTAMPTZ NULL,
    revoked_at    TIMESTAMPTZ NULL,
    view_count    INTEGER     NOT NULL DEFAULT 0,
    created_by    UUID        NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_share_links_file ON document_share_links (tenant_id, file_id, created_at DESC);

CALL enable_tenant_rls('document_share_links');

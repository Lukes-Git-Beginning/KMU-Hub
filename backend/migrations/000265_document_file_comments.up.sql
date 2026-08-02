-- ============================================================================
-- document_file_comments: comments on document files (backend-gaps.md
-- §dokumente "Datei-Kommentare"). Author-only edit, author-or-admin delete
-- (enforced in internal/document/file, not via RLS).
-- ============================================================================

CREATE TABLE document_file_comments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    file_id     UUID        NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    author_id   UUID        NOT NULL REFERENCES users(id),
    content     TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_document_file_comments_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_document_file_comments_file   ON document_file_comments(file_id, created_at);
CREATE INDEX idx_document_file_comments_tenant ON document_file_comments(tenant_id, created_at DESC);

CALL enable_tenant_rls('document_file_comments');

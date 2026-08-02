-- ============================================================================
-- document_file_activity: append-only audit trail for document file mutations
-- (uploaded, renamed, moved, copied, downloaded, shared, version_created,
-- reverted). Backs GET /api/v1/documents/files/{id}/activity.
-- ============================================================================

CREATE TABLE document_file_activity (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    file_id     UUID        NOT NULL REFERENCES document_files(id) ON DELETE CASCADE,
    action      VARCHAR(20) NOT NULL CHECK (action IN (
                    'uploaded', 'renamed', 'moved', 'copied', 'downloaded',
                    'shared', 'version_created', 'reverted'
                )),
    actor_id    UUID        NOT NULL REFERENCES users(id),
    detail      TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_document_file_activity_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_document_file_activity_file   ON document_file_activity(file_id, created_at DESC);
CREATE INDEX idx_document_file_activity_tenant ON document_file_activity(tenant_id, created_at DESC);

CALL enable_tenant_rls('document_file_activity');

-- Append-only enforcement: no application bug or direct DB access can alter
-- a file's audit trail after the fact (same pattern as audit_log, mig 000222).
CREATE OR REPLACE FUNCTION prevent_document_file_activity_mutation() RETURNS trigger
    LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'document_file_activity is append-only';
END;
$$;

CREATE TRIGGER document_file_activity_append_only
    BEFORE UPDATE OR DELETE ON document_file_activity
    FOR EACH ROW EXECUTE FUNCTION prevent_document_file_activity_mutation();

-- Migration 000139 — GoBD Belegarchiv (Revisionssichere Beleg-Ablage, §147 AO)
--
-- Implements an immutable document archive for GoBD-compliant storage of
-- Belege (receipts, invoices, contracts, correspondence) as required by §147 AO
-- (10-year retention, Buchführungspflicht).
--
-- Design decisions:
-- - NO UPDATE/DELETE on gobd_documents (immutability by design — service-side)
-- - retention_until = 31.12 of (archived_year + 8), giving ≥10 complete
--   calendar years of retention from the first possible archival date in that year.
-- - sha256: stored as CHAR(64) hex for fast integrity checks without decoding
-- - Events log is append-only: archived, annotation, access, integrity_check
-- - source_invoice_id: nullable, links documents to finance_invoices for traceability
-- - CALL enable_tenant_rls on both tables for RLS isolation

BEGIN;

SET LOCAL row_security = off;

-- ============================================================================
-- ENUMs
-- ============================================================================

CREATE TYPE gobd_document_type AS ENUM (
    'invoice',
    'credit_note',
    'receipt',
    'contract',
    'correspondence',
    'other'
);

CREATE TYPE gobd_event_type AS ENUM (
    'archived',
    'annotation',
    'access',
    'integrity_check'
);

-- ============================================================================
-- gobd_documents — the immutable archive
-- ============================================================================

CREATE TABLE gobd_documents (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID         NOT NULL,
    doc_type          gobd_document_type NOT NULL DEFAULT 'other',
    source_invoice_id UUID         NULL REFERENCES finance_invoices(id) ON DELETE SET NULL,
    storage_key       TEXT         NOT NULL,
    sha256            CHAR(64)     NOT NULL CHECK (LENGTH(sha256) = 64),
    original_filename TEXT         NOT NULL,
    mime_type         TEXT         NOT NULL,
    file_size_bytes   BIGINT       NOT NULL CHECK (file_size_bytes > 0),
    archived_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    archived_by       UUID         NOT NULL,
    retention_until   DATE         NOT NULL,

    CONSTRAINT fk_gobd_documents_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_gobd_documents_tenant            ON gobd_documents(tenant_id);
CREATE INDEX idx_gobd_documents_source_invoice    ON gobd_documents(source_invoice_id) WHERE source_invoice_id IS NOT NULL;
CREATE INDEX idx_gobd_documents_tenant_archived   ON gobd_documents(tenant_id, archived_at DESC);
CREATE INDEX idx_gobd_documents_tenant_doc_type   ON gobd_documents(tenant_id, doc_type);
CREATE INDEX idx_gobd_documents_retention         ON gobd_documents(retention_until);

-- ============================================================================
-- gobd_document_events — audit trail (append-only)
-- ============================================================================

CREATE TABLE gobd_document_events (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID            NOT NULL,
    document_id UUID            NOT NULL REFERENCES gobd_documents(id) ON DELETE CASCADE,
    event_type  gobd_event_type NOT NULL,
    created_by  UUID            NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    note        TEXT            NOT NULL DEFAULT '',
    metadata    JSONB           NOT NULL DEFAULT '{}',

    CONSTRAINT fk_gobd_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_gobd_events_document    ON gobd_document_events(document_id, created_at DESC);
CREATE INDEX idx_gobd_events_tenant      ON gobd_document_events(tenant_id, created_at DESC);

-- ============================================================================
-- Enable RLS on both tables
-- ============================================================================

CALL enable_tenant_rls('gobd_documents');
CALL enable_tenant_rls('gobd_document_events');

-- ============================================================================
-- RBAC — permission seeds
-- ============================================================================
-- gobd-archive:read   — list and view archived documents
-- gobd-archive:write  — archive new documents
-- gobd-archive:admin  — access log review + annotation management

INSERT INTO permissions (name, resource, action) VALUES
    ('gobd-archive:read',  'gobd-archive', 'read'),
    ('gobd-archive:write', 'gobd-archive', 'write'),
    ('gobd-archive:admin', 'gobd-archive', 'admin')
ON CONFLICT (name) DO NOTHING;

-- admin: all three permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
  AND p.name IN ('gobd-archive:read', 'gobd-archive:write', 'gobd-archive:admin')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- manager: read + write (not admin)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'manager'
  AND p.name IN ('gobd-archive:read', 'gobd-archive:write')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- member: read only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'member'
  AND p.name = 'gobd-archive:read'
ON CONFLICT (role_id, permission_id) DO NOTHING;

COMMIT;

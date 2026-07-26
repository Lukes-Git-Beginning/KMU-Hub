-- Migration 000245: report_documents
-- Backend persistence for the multi-page report authoring documents
-- ("Schicht 4"), distinct from report_definitions ("Schicht 3", a single
-- chart/data widget). berichte-client.ts already calls
-- /api/v1/berichte/documents{,/:id}; until now nothing served it.
--
-- rows/settings are stored as opaque JSONB: the block tree
-- (rows -> columns -> blocks, 13 block types, TipTap HTML inside text blocks)
-- is owned by the frontend, and its keys are camelCase. The service validates
-- shape (array/object) and size only, never block semantics.
-- lean: JSONB passthrough without per-block-type validation; add server-side
-- block validation when a second client (server PDF export) starts depending
-- on the block structure being well-formed.

BEGIN;

CREATE TABLE report_documents (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    module      TEXT        NOT NULL DEFAULT 'cross',
    status      TEXT        NOT NULL DEFAULT 'draft',
    "rows"      JSONB       NOT NULL DEFAULT '[]'::jsonb,
    settings    JSONB       NOT NULL DEFAULT '{}'::jsonb,
    -- Free-form template key. Templates are frontend-side starter structures
    -- (no report_templates table exists), so this is deliberately not a FK.
    template_id TEXT        NULL,
    created_by  UUID        NULL REFERENCES users (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ NULL,
    CONSTRAINT report_documents_module_check
        CHECK (module IN ('finanzen', 'crm', 'helpdesk', 'inventar', 'produktion', 'cross')),
    CONSTRAINT report_documents_status_check
        CHECK (status IN ('draft', 'final', 'released', 'archived'))
);

-- Library listing: tenant-scoped, newest first (matches the default sort).
CREATE INDEX idx_report_documents_tenant_updated
    ON report_documents (tenant_id, updated_at DESC);
CREATE INDEX idx_report_documents_tenant_module
    ON report_documents (tenant_id, module);
CREATE INDEX idx_report_documents_tenant_status
    ON report_documents (tenant_id, status);

-- RLS: standard tenant isolation (matches pattern in 000238).
ALTER TABLE report_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_documents FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON report_documents
    USING      (tenant_id = current_tenant_id() OR is_system_context())
    WITH CHECK (tenant_id = current_tenant_id() OR is_system_context());

COMMIT;

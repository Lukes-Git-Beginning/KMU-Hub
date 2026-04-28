-- Sprint 2 W2.A Rapporte — schema

-- ============================================================================
-- work_reports
-- ============================================================================
CREATE TABLE work_reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'approved', 'rejected')),
    author_id    UUID NOT NULL,
    reviewer_id  UUID NULL,
    reviewed_at  TIMESTAMPTZ NULL,
    review_note  TEXT NOT NULL DEFAULT '',
    lat          NUMERIC(9,6) NULL,
    lon          NUMERIC(9,6) NULL,
    report_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ NULL
);

CREATE INDEX idx_work_reports_tenant_id     ON work_reports (tenant_id);
CREATE INDEX idx_work_reports_tenant_active ON work_reports (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_reports_status        ON work_reports (tenant_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_reports_author        ON work_reports (tenant_id, author_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_work_reports_pending       ON work_reports (tenant_id) WHERE status = 'submitted' AND deleted_at IS NULL;

-- ============================================================================
-- report_lines
-- ============================================================================
CREATE TABLE report_lines (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    report_id   UUID NOT NULL REFERENCES work_reports (id) ON DELETE CASCADE,
    position    INT NOT NULL DEFAULT 0,
    description TEXT NOT NULL,
    quantity    NUMERIC(12,4) NOT NULL DEFAULT 0,
    unit        TEXT NOT NULL DEFAULT 'h',
    notes       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_lines_tenant_id ON report_lines (tenant_id);
CREATE INDEX idx_report_lines_report_id ON report_lines (report_id);

-- ============================================================================
-- report_attachments
-- ============================================================================
CREATE TABLE report_attachments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    report_id   UUID NOT NULL REFERENCES work_reports (id) ON DELETE CASCADE,
    line_id     UUID NULL REFERENCES report_lines (id) ON DELETE SET NULL,
    filename    TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    object_key  TEXT NOT NULL,
    uploaded_by UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_attachments_tenant_id ON report_attachments (tenant_id);
CREATE INDEX idx_report_attachments_report_id ON report_attachments (report_id);
CREATE INDEX idx_report_attachments_line_id   ON report_attachments (line_id) WHERE line_id IS NOT NULL;

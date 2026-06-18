-- Migration 000162: Extend work_reports + report_lines with new fields,
-- add report_workers table.

BEGIN;

-- ============================================================================
-- work_reports: new field-report columns
-- ============================================================================
ALTER TABLE work_reports
    ADD COLUMN IF NOT EXISTS weather        TEXT,
    ADD COLUMN IF NOT EXISTS temperature    NUMERIC(5,2),
    ADD COLUMN IF NOT EXISTS work_start     TIME,
    ADD COLUMN IF NOT EXISTS work_end       TIME,
    ADD COLUMN IF NOT EXISTS break_minutes  INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS project_name   TEXT;

-- ============================================================================
-- report_lines: new categorisation columns
-- ============================================================================
ALTER TABLE report_lines
    ADD COLUMN IF NOT EXISTS category TEXT,
    ADD COLUMN IF NOT EXISTS article  TEXT;

-- ============================================================================
-- report_workers: workers assigned to a report
-- ============================================================================
CREATE TABLE IF NOT EXISTS report_workers (
    id         UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id  UUID        NOT NULL,
    report_id  UUID        NOT NULL REFERENCES work_reports(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    role       TEXT,
    hours      NUMERIC(5,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CALL enable_tenant_rls('report_workers');

CREATE INDEX IF NOT EXISTS idx_report_workers_report_id  ON report_workers(report_id);
CREATE INDEX IF NOT EXISTS idx_report_workers_tenant_id  ON report_workers(tenant_id);

COMMIT;

-- Migration 000163: Add measurements, measurement_positions, report_templates tables.

BEGIN;

-- ============================================================================
-- measurements
-- ============================================================================
CREATE TABLE IF NOT EXISTS measurements (
    id          UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id   UUID        NOT NULL,
    report_id   UUID        REFERENCES work_reports(id) ON DELETE SET NULL,
    title       TEXT        NOT NULL,
    location    TEXT,
    measured_by TEXT,
    measured_at DATE,
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CALL enable_tenant_rls('measurements');

CREATE INDEX IF NOT EXISTS idx_measurements_tenant_id  ON measurements(tenant_id);
CREATE INDEX IF NOT EXISTS idx_measurements_report_id  ON measurements(report_id);

-- ============================================================================
-- measurement_positions
-- ============================================================================
CREATE TABLE IF NOT EXISTS measurement_positions (
    id              UUID         NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id       UUID         NOT NULL,
    measurement_id  UUID         NOT NULL REFERENCES measurements(id) ON DELETE CASCADE,
    position_number INTEGER      NOT NULL DEFAULT 1,
    description     TEXT         NOT NULL,
    unit            TEXT,
    quantity        NUMERIC(12,4),
    unit_price      NUMERIC(12,2),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CALL enable_tenant_rls('measurement_positions');

CREATE INDEX IF NOT EXISTS idx_measurement_positions_tenant_id      ON measurement_positions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_measurement_positions_measurement_id ON measurement_positions(measurement_id);

-- ============================================================================
-- report_templates
-- ============================================================================
CREATE TABLE IF NOT EXISTS report_templates (
    id            UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id     UUID        NOT NULL,
    name          TEXT        NOT NULL,
    description   TEXT,
    category      TEXT,
    default_lines JSONB       NOT NULL DEFAULT '[]',
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CALL enable_tenant_rls('report_templates');

CREATE INDEX IF NOT EXISTS idx_report_templates_tenant_id ON report_templates(tenant_id);

COMMIT;

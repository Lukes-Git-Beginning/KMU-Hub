-- Sprint 2 W2.A Schichten — schema

-- ============================================================================
-- shifts
-- ============================================================================
CREATE TABLE shifts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    start_time  TIMESTAMPTZ NOT NULL,
    end_time    TIMESTAMPTZ NOT NULL,
    status      TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    location    TEXT NULL,
    created_by  UUID NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shifts_tenant_id       ON shifts (tenant_id);
CREATE INDEX idx_shifts_tenant_status   ON shifts (tenant_id, status);
CREATE INDEX idx_shifts_tenant_period   ON shifts (tenant_id, start_time, end_time);
CREATE INDEX idx_shifts_published       ON shifts (tenant_id) WHERE status = 'published';

-- ============================================================================
-- shift_assignments
-- ============================================================================
CREATE TABLE shift_assignments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    shift_id    UUID NOT NULL REFERENCES shifts (id) ON DELETE CASCADE,
    employee_id UUID NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID NULL,
    UNIQUE (shift_id, employee_id)
);

CREATE INDEX idx_shift_assignments_tenant_id   ON shift_assignments (tenant_id);
CREATE INDEX idx_shift_assignments_shift_id    ON shift_assignments (shift_id);
CREATE INDEX idx_shift_assignments_employee_id ON shift_assignments (employee_id);

-- ============================================================================
-- shift_templates
-- ============================================================================
CREATE TABLE shift_templates (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    -- day_of_week: 0=Sunday … 6=Saturday (ISO weekday mapped)
    day_of_week  INT NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_hour   INT NOT NULL CHECK (start_hour >= 0 AND start_hour <= 23),
    start_minute INT NOT NULL CHECK (start_minute >= 0 AND start_minute <= 59),
    duration_minutes INT NOT NULL CHECK (duration_minutes > 0),
    location     TEXT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shift_templates_tenant_id ON shift_templates (tenant_id);

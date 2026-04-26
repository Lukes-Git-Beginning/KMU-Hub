-- Sprint 2 W1.A Produktion — schema

-- ============================================================================
-- production_orders
-- ============================================================================
CREATE TABLE production_orders (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    order_number  TEXT        NOT NULL,
    product_name  TEXT        NOT NULL,
    quantity      INT         NOT NULL CHECK (quantity > 0),
    status        TEXT        NOT NULL DEFAULT 'planned'
                                CHECK (status IN ('planned','in_progress','completed','cancelled')),
    planned_start TIMESTAMPTZ NOT NULL,
    planned_end   TIMESTAMPTZ NOT NULL,
    actual_start  TIMESTAMPTZ NULL,
    actual_end    TIMESTAMPTZ NULL,
    priority      INT         NOT NULL DEFAULT 3 CHECK (priority BETWEEN 1 AND 5),
    notes         TEXT        NOT NULL DEFAULT '',
    created_by    UUID        NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, order_number)
);

CREATE INDEX idx_production_orders_tenant_status ON production_orders (tenant_id, status);
CREATE INDEX idx_production_orders_planned_start  ON production_orders (tenant_id, planned_start);

-- ============================================================================
-- machine_bookings
-- ============================================================================
CREATE TABLE machine_bookings (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    machine_id           TEXT        NOT NULL,
    production_order_id  UUID        NOT NULL REFERENCES production_orders (id) ON DELETE CASCADE,
    starts_at            TIMESTAMPTZ NOT NULL,
    ends_at              TIMESTAMPTZ NOT NULL,
    status               TEXT        NOT NULL DEFAULT 'booked'
                                       CHECK (status IN ('booked','in_use','completed','cancelled')),
    notes                TEXT        NOT NULL DEFAULT '',
    created_by           UUID        NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);

-- Composite index used by the conflict-check query.
CREATE INDEX idx_machine_bookings_machine_time ON machine_bookings (tenant_id, machine_id, starts_at, ends_at);
CREATE INDEX idx_machine_bookings_order_id      ON machine_bookings (production_order_id);

-- ============================================================================
-- production_plans
-- ============================================================================
CREATE TABLE production_plans (
    id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    name                  TEXT         NOT NULL,
    week_number           INT          NOT NULL CHECK (week_number BETWEEN 1 AND 53),
    year                  INT          NOT NULL,
    total_capacity_hours  NUMERIC(8,2) NOT NULL DEFAULT 0,
    planned_capacity_hours NUMERIC(8,2) NOT NULL DEFAULT 0,
    status                TEXT         NOT NULL DEFAULT 'draft'
                                         CHECK (status IN ('draft','active','completed')),
    notes                 TEXT         NOT NULL DEFAULT '',
    created_by            UUID         NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_production_plans_tenant_week ON production_plans (tenant_id, year, week_number);

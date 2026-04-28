-- Sprint 2 W2.A Vermietung — schema

-- ============================================================================
-- rental_objects (the things that can be rented out)
-- ============================================================================
CREATE TABLE rental_objects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    name        TEXT NOT NULL,
    description TEXT NULL,
    category    TEXT NOT NULL DEFAULT '',
    daily_rate  NUMERIC(12,2) NOT NULL DEFAULT 0,
    deposit     NUMERIC(12,2) NOT NULL DEFAULT 0,
    location    TEXT NULL,
    notes       TEXT NULL,
    active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ NULL
);

CREATE INDEX idx_rental_objects_tenant_id     ON rental_objects (tenant_id);
CREATE INDEX idx_rental_objects_tenant_active ON rental_objects (tenant_id) WHERE deleted_at IS NULL;

-- ============================================================================
-- rentals
-- ============================================================================
CREATE TABLE rentals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    object_id   UUID NOT NULL REFERENCES rental_objects (id) ON DELETE CASCADE,
    contact_id  UUID NULL,                                -- FK into crm contacts (soft ref, no cascade)
    renter_name TEXT NOT NULL DEFAULT '',
    start_date  TIMESTAMPTZ NOT NULL,
    end_date    TIMESTAMPTZ NOT NULL,
    status      TEXT NOT NULL DEFAULT 'reserved'
                    CHECK (status IN ('reserved', 'active', 'completed', 'cancelled')),
    total_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    deposit_paid BOOLEAN NOT NULL DEFAULT FALSE,
    notes       TEXT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT rentals_dates_check CHECK (end_date > start_date)
);

-- GIST index for fast date-range overlap detection (critical path)
CREATE INDEX idx_rentals_object_dates ON rentals
    USING GIST (object_id, tstzrange(start_date, end_date));

CREATE INDEX idx_rentals_tenant_id   ON rentals (tenant_id);
CREATE INDEX idx_rentals_object_id   ON rentals (object_id);
CREATE INDEX idx_rentals_status      ON rentals (tenant_id, status);
CREATE INDEX idx_rentals_dates       ON rentals (tenant_id, start_date, end_date);

-- ============================================================================
-- rental_inspections (handover / return condition records)
-- ============================================================================
CREATE TABLE rental_inspections (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    rental_id  UUID NOT NULL REFERENCES rentals (id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('handover', 'return')),
    notes      TEXT NOT NULL DEFAULT '',
    photo_urls TEXT[] NOT NULL DEFAULT '{}',
    performed_by UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rental_inspections_tenant_id ON rental_inspections (tenant_id);
CREATE INDEX idx_rental_inspections_rental_id ON rental_inspections (rental_id);

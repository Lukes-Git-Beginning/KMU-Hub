-- Sprint 2 W2.A Fuhrpark — schema

-- ============================================================================
-- vehicles
-- ============================================================================
CREATE TABLE vehicles (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    license_plate          TEXT NOT NULL,
    make                   TEXT NOT NULL,
    model                  TEXT NOT NULL,
    year                   INT NOT NULL,
    vin                    TEXT NULL,
    color                  TEXT NULL,
    fuel_type              TEXT NOT NULL DEFAULT 'petrol'
                               CHECK (fuel_type IN ('petrol', 'diesel', 'electric', 'hybrid', 'other')),
    status                 TEXT NOT NULL DEFAULT 'active'
                               CHECK (status IN ('active', 'in_service', 'inactive', 'decommissioned')),
    mileage_km             BIGINT NOT NULL DEFAULT 0,
    tuev_due_date          DATE NULL,
    tuev_reminder_sent_at  TIMESTAMPTZ NULL,
    assigned_driver_id     UUID NULL,   -- stub: Sprint 3 team-wiring
    notes                  TEXT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at             TIMESTAMPTZ NULL,
    UNIQUE (tenant_id, license_plate)
);

CREATE INDEX idx_vehicles_tenant_id     ON vehicles (tenant_id);
CREATE INDEX idx_vehicles_tenant_active ON vehicles (tenant_id) WHERE deleted_at IS NULL;
-- Partial index used by TUEV cron to quickly scan upcoming due dates
CREATE INDEX idx_vehicles_tuev_due      ON vehicles (tenant_id, tuev_due_date)
    WHERE tuev_due_date IS NOT NULL AND deleted_at IS NULL;

-- ============================================================================
-- vehicle_services
-- ============================================================================
CREATE TABLE vehicle_services (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    vehicle_id    UUID NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    service_type  TEXT NOT NULL,
    description   TEXT NULL,
    scheduled_at  DATE NOT NULL,
    completed_at  TIMESTAMPTZ NULL,
    cost_cents    BIGINT NULL,
    workshop      TEXT NULL,
    mileage_km    BIGINT NULL,
    status        TEXT NOT NULL DEFAULT 'scheduled'
                      CHECK (status IN ('scheduled', 'in_progress', 'completed', 'cancelled')),
    notes         TEXT NULL,
    created_by    UUID NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vehicle_services_tenant_id  ON vehicle_services (tenant_id);
CREATE INDEX idx_vehicle_services_vehicle_id ON vehicle_services (vehicle_id);
CREATE INDEX idx_vehicle_services_scheduled  ON vehicle_services (vehicle_id, scheduled_at DESC);

-- ============================================================================
-- vehicle_damages
-- ============================================================================
CREATE TABLE vehicle_damages (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    vehicle_id    UUID NOT NULL REFERENCES vehicles (id) ON DELETE CASCADE,
    description   TEXT NOT NULL,
    severity      TEXT NOT NULL DEFAULT 'minor'
                      CHECK (severity IN ('minor', 'moderate', 'major', 'totalled')),
    status        TEXT NOT NULL DEFAULT 'reported'
                      CHECK (status IN ('reported', 'in_repair', 'resolved')),
    reported_by   UUID NULL,
    resolved_by   UUID NULL,
    resolved_at   TIMESTAMPTZ NULL,
    photo_keys    TEXT[] NOT NULL DEFAULT '{}',  -- MinIO object keys
    cost_cents    BIGINT NULL,
    notes         TEXT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_vehicle_damages_tenant_id  ON vehicle_damages (tenant_id);
CREATE INDEX idx_vehicle_damages_vehicle_id ON vehicle_damages (vehicle_id);
CREATE INDEX idx_vehicle_damages_status     ON vehicle_damages (vehicle_id, status);

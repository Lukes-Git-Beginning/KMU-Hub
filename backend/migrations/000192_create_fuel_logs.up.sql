CREATE TABLE IF NOT EXISTS fuel_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    vehicle_id  UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    date        DATE NOT NULL,
    liters      NUMERIC(8,2) NOT NULL CHECK (liters > 0),
    cost_cents  BIGINT NOT NULL CHECK (cost_cents >= 0),
    mileage_km  INTEGER NOT NULL CHECK (mileage_km >= 0),
    fuel_type   TEXT NOT NULL DEFAULT 'diesel' CHECK (fuel_type IN ('diesel','petrol','electric','hybrid','other')),
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_fuel_logs_vehicle_id ON fuel_logs(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_fuel_logs_tenant_id ON fuel_logs(tenant_id);
CALL enable_tenant_rls('fuel_logs');

CREATE TABLE IF NOT EXISTS trip_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    vehicle_id      UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    date            DATE NOT NULL,
    start_location  TEXT NOT NULL,
    end_location    TEXT NOT NULL,
    purpose         TEXT NOT NULL,
    start_km        INTEGER NOT NULL CHECK (start_km >= 0),
    end_km          INTEGER NOT NULL CHECK (end_km >= 0),
    km              INTEGER GENERATED ALWAYS AS (end_km - start_km) STORED,
    is_private      BOOLEAN NOT NULL DEFAULT FALSE,
    driver_name     TEXT NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT trip_logs_km_range_check CHECK (end_km >= start_km)
);
CREATE INDEX IF NOT EXISTS idx_trip_logs_vehicle_id ON trip_logs(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_trip_logs_tenant_id ON trip_logs(tenant_id);
CALL enable_tenant_rls('trip_logs');

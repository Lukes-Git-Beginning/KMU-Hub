CREATE TABLE IF NOT EXISTS gps_positions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    vehicle_id  UUID NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    lat         DOUBLE PRECISION NOT NULL,
    lng         DOUBLE PRECISION NOT NULL,
    speed_kmh   NUMERIC(6,2),
    recorded_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_gps_positions_vehicle_id_recorded_at ON gps_positions(vehicle_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_gps_positions_tenant_id ON gps_positions(tenant_id);
CALL enable_tenant_rls('gps_positions');

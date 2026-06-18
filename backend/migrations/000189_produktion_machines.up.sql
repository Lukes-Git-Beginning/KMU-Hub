-- 000189 Produktion Maschinen (verwaltete Entitäten)

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS production_machines (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    name        TEXT        NOT NULL,
    type        TEXT        NOT NULL DEFAULT '',
    status      TEXT        NOT NULL DEFAULT 'available' CHECK (status IN ('available','in_use','maintenance')),
    notes       TEXT        NOT NULL DEFAULT '',
    created_by  UUID        NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_production_machines_tenant
    ON production_machines (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_production_machines_status
    ON production_machines (tenant_id, status);

CALL enable_tenant_rls('production_machines');

COMMIT;

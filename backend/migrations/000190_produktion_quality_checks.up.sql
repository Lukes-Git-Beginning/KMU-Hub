-- 000190 Produktion Qualitätsprüfungen

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS production_quality_checks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    order_id        UUID        NOT NULL REFERENCES production_orders(id) ON DELETE CASCADE,
    inspector       TEXT        NOT NULL,
    checked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    passed          BOOLEAN     NOT NULL DEFAULT true,
    defects_found   INTEGER     NOT NULL DEFAULT 0,
    notes           TEXT        NOT NULL DEFAULT '',
    created_by      UUID        NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_production_quality_checks_order
    ON production_quality_checks (order_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_production_quality_checks_tenant
    ON production_quality_checks (tenant_id, created_at DESC);

CALL enable_tenant_rls('production_quality_checks');

COMMIT;

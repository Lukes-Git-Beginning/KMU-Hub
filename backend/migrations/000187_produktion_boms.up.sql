-- 000187 Produktion BOMs (Stücklisten)

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS production_boms (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL,
    product_name TEXT        NOT NULL,
    sku         TEXT         NOT NULL,
    version     TEXT         NOT NULL DEFAULT '1.0',
    active      BOOLEAN      NOT NULL DEFAULT true,
    notes       TEXT         NOT NULL DEFAULT '',
    created_by  UUID         NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_production_boms_tenant
    ON production_boms (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_production_boms_active
    ON production_boms (tenant_id, active)
    WHERE active = true;
CREATE UNIQUE INDEX IF NOT EXISTS idx_production_boms_tenant_sku
    ON production_boms (tenant_id, sku);

CREATE TABLE IF NOT EXISTS production_bom_items (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    bom_id          UUID        NOT NULL REFERENCES production_boms(id) ON DELETE CASCADE,
    material_name   TEXT        NOT NULL,
    quantity        NUMERIC(12,4) NOT NULL DEFAULT 1,
    unit            TEXT        NOT NULL DEFAULT 'Stk',
    sort_order      INTEGER     NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_production_bom_items_bom
    ON production_bom_items (bom_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_production_bom_items_tenant
    ON production_bom_items (tenant_id);

-- Optional: FK on production_orders for bom reference (nullable, no breaking change)
ALTER TABLE production_orders
    ADD COLUMN IF NOT EXISTS bom_id UUID NULL REFERENCES production_boms(id) ON DELETE SET NULL;

CALL enable_tenant_rls('production_boms');
CALL enable_tenant_rls('production_bom_items');

COMMIT;

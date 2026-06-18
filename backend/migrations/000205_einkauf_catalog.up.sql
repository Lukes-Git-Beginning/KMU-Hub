-- 000205 Einkauf Supplier Catalog Items

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS supplier_catalog_items (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID          NOT NULL,
    supplier_id     UUID          NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    name            TEXT          NOT NULL,
    sku             TEXT          NOT NULL DEFAULT '',
    category        TEXT          NOT NULL DEFAULT '',
    price           NUMERIC(15,4) NOT NULL DEFAULT 0,
    currency        TEXT          NOT NULL DEFAULT 'EUR',
    unit            TEXT          NOT NULL DEFAULT 'Stk',
    available       BOOLEAN       NOT NULL DEFAULT true,
    min_order_qty   NUMERIC(15,4) NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_supplier_catalog_tenant ON supplier_catalog_items (tenant_id, supplier_id);

CALL enable_tenant_rls('supplier_catalog_items');

COMMIT;

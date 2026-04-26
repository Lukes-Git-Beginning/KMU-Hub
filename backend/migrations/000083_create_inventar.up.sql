-- Sprint 2 W1.A Inventar — schema

-- ============================================================================
-- inventory_items
-- ============================================================================
CREATE TABLE inventory_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    name         TEXT NOT NULL,
    sku          TEXT NOT NULL,
    barcode      TEXT NULL,
    quantity     BIGINT NOT NULL DEFAULT 0,
    min_quantity BIGINT NOT NULL DEFAULT 0,
    unit         TEXT NOT NULL DEFAULT 'Stk',
    location     TEXT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ NULL,
    UNIQUE (tenant_id, sku)
);

CREATE INDEX idx_inventory_items_tenant_id     ON inventory_items (tenant_id);
CREATE INDEX idx_inventory_items_tenant_active ON inventory_items (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_inventory_items_sku           ON inventory_items (tenant_id, sku) WHERE deleted_at IS NULL;

-- ============================================================================
-- inventory_movements
-- ============================================================================
CREATE TABLE inventory_movements (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    item_id       UUID NOT NULL REFERENCES inventory_items (id) ON DELETE CASCADE,
    movement_type TEXT NOT NULL CHECK (movement_type IN ('in', 'out', 'adjustment', 'transfer')),
    quantity      BIGINT NOT NULL,
    performed_by  UUID NULL,
    reason        TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_movements_tenant_id ON inventory_movements (tenant_id);
CREATE INDEX idx_inventory_movements_item_id   ON inventory_movements (item_id);
CREATE INDEX idx_inventory_movements_created   ON inventory_movements (item_id, created_at DESC);

-- ============================================================================
-- stock_warnings
-- ============================================================================
CREATE TABLE stock_warnings (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    item_id          UUID NOT NULL REFERENCES inventory_items (id) ON DELETE CASCADE,
    threshold        BIGINT NOT NULL DEFAULT 0,
    current_quantity BIGINT NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    acknowledged_at  TIMESTAMPTZ NULL,
    acknowledged_by  UUID NULL
);

CREATE INDEX idx_stock_warnings_tenant_id ON stock_warnings (tenant_id);
CREATE INDEX idx_stock_warnings_item_id   ON stock_warnings (item_id);
CREATE INDEX idx_stock_warnings_active    ON stock_warnings (tenant_id, item_id) WHERE status = 'active';

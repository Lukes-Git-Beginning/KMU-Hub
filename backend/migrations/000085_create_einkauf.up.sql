-- Sprint 2 W1.A Einkauf — schema
-- Wareneingangs-Sync mit inventar ist Sprint-3-Item; received_quantity wird
-- vorerst nur lokal getrackt (kein Cross-Modul-Aufruf).

-- ============================================================================
-- suppliers
-- ============================================================================
CREATE TABLE suppliers (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    name          TEXT        NOT NULL,
    contact_id    UUID        NULL, -- FK to contacts ON DELETE SET NULL — contacts table must exist
    email         TEXT        NOT NULL DEFAULT '',
    phone         TEXT        NOT NULL DEFAULT '',
    address       TEXT        NOT NULL DEFAULT '',
    payment_terms TEXT        NOT NULL DEFAULT '',
    notes         TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ NULL
);

CREATE INDEX idx_suppliers_tenant_id ON suppliers (tenant_id);
CREATE INDEX idx_suppliers_name      ON suppliers (tenant_id, LOWER(name));

-- ============================================================================
-- purchase_orders
-- ============================================================================
CREATE TABLE purchase_orders (
    id                     UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    supplier_id            UUID        NOT NULL REFERENCES suppliers (id) ON DELETE CASCADE,
    po_number              TEXT        NOT NULL,
    status                 TEXT        NOT NULL DEFAULT 'draft'
                               CHECK (status IN ('draft','submitted','sent','partially_received','received','closed','cancelled')),
    order_date             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expected_delivery_date TIMESTAMPTZ NULL,
    total_amount           NUMERIC(15,4) NOT NULL DEFAULT 0,
    currency               TEXT        NOT NULL DEFAULT 'EUR',
    notes                  TEXT        NOT NULL DEFAULT '',
    created_by             UUID        NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, po_number)
);

CREATE INDEX idx_purchase_orders_tenant_status ON purchase_orders (tenant_id, status);
CREATE INDEX idx_purchase_orders_supplier_id   ON purchase_orders (supplier_id);
CREATE INDEX idx_purchase_orders_order_date    ON purchase_orders (tenant_id, order_date DESC);

-- ============================================================================
-- po_lines
-- ============================================================================
CREATE TABLE po_lines (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    po_id             UUID        NOT NULL REFERENCES purchase_orders (id) ON DELETE CASCADE,
    product_name      TEXT        NOT NULL,
    sku               TEXT        NOT NULL DEFAULT '',
    quantity          NUMERIC(15,4) NOT NULL,
    unit_price        NUMERIC(15,4) NOT NULL,
    tax_rate          NUMERIC(5,4)  NOT NULL DEFAULT 0,
    received_quantity NUMERIC(15,4) NOT NULL DEFAULT 0,
    line_position     INT         NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_po_lines_po_id     ON po_lines (po_id);
CREATE INDEX idx_po_lines_tenant_id ON po_lines (tenant_id);

-- 000207 Einkauf Framework Contracts + Items + Calls

BEGIN;
SET LOCAL row_security = off;

CREATE TABLE IF NOT EXISTS framework_contracts (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID          NOT NULL,
    supplier_id  UUID          NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    title        TEXT          NOT NULL,
    contract_nr  TEXT          NOT NULL,
    start_date   DATE,
    end_date     DATE,
    total_value  NUMERIC(15,4) NOT NULL DEFAULT 0,
    used_value   NUMERIC(15,4) NOT NULL DEFAULT 0,
    currency     TEXT          NOT NULL DEFAULT 'EUR',
    status       TEXT          NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'expired')),
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, contract_nr)
);

CREATE INDEX IF NOT EXISTS idx_framework_contracts_tenant ON framework_contracts (tenant_id, supplier_id);

CALL enable_tenant_rls('framework_contracts');

-- -------------------------------------------------------

CREATE TABLE IF NOT EXISTS framework_contract_items (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID          NOT NULL,
    contract_id  UUID          NOT NULL REFERENCES framework_contracts(id) ON DELETE CASCADE,
    name         TEXT          NOT NULL,
    unit_price   NUMERIC(15,4) NOT NULL DEFAULT 0,
    unit         TEXT          NOT NULL DEFAULT 'Stk',
    agreed_qty   NUMERIC(15,4) NOT NULL DEFAULT 0,
    called_qty   NUMERIC(15,4) NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_framework_contract_items_tenant ON framework_contract_items (tenant_id, contract_id);

CALL enable_tenant_rls('framework_contract_items');

-- -------------------------------------------------------

CREATE TABLE IF NOT EXISTS framework_contract_calls (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID          NOT NULL,
    contract_id UUID          NOT NULL REFERENCES framework_contracts(id) ON DELETE CASCADE,
    po_id       UUID          NULL REFERENCES purchase_orders(id) ON DELETE SET NULL,
    amount      NUMERIC(15,4) NOT NULL DEFAULT 0,
    currency    TEXT          NOT NULL DEFAULT 'EUR',
    called_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    notes       TEXT          NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_framework_contract_calls_tenant ON framework_contract_calls (tenant_id, contract_id);

CALL enable_tenant_rls('framework_contract_calls');

COMMIT;

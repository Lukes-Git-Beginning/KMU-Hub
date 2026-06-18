-- 000208 Einkauf — add optional framework_contract_id to purchase_orders

BEGIN;

ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS framework_contract_id UUID NULL
        REFERENCES framework_contracts(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_po_framework_contract
    ON purchase_orders (framework_contract_id)
    WHERE framework_contract_id IS NOT NULL;

COMMIT;

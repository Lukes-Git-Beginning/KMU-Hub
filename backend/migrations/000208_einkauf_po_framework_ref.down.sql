-- 000208 Einkauf — revert framework_contract_id on purchase_orders

BEGIN;

DROP INDEX IF EXISTS idx_po_framework_contract;

ALTER TABLE purchase_orders
    DROP COLUMN IF EXISTS framework_contract_id;

COMMIT;

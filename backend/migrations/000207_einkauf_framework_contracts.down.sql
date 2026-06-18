-- 000207 Einkauf Framework Contracts + Items + Calls — revert
-- Drop in reverse dependency order

BEGIN;

DROP TABLE IF EXISTS framework_contract_calls;
DROP TABLE IF EXISTS framework_contract_items;
DROP TABLE IF EXISTS framework_contracts;

COMMIT;

BEGIN;
ALTER TABLE production_orders DROP COLUMN IF EXISTS bom_id;
DROP TABLE IF EXISTS production_bom_items;
DROP TABLE IF EXISTS production_boms;
COMMIT;

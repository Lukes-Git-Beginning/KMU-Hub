-- Sprint 2 W1.A Inventar — rollback

DROP INDEX IF EXISTS idx_stock_warnings_active;
DROP INDEX IF EXISTS idx_stock_warnings_item_id;
DROP INDEX IF EXISTS idx_stock_warnings_tenant_id;
DROP TABLE IF EXISTS stock_warnings;

DROP INDEX IF EXISTS idx_inventory_movements_created;
DROP INDEX IF EXISTS idx_inventory_movements_item_id;
DROP INDEX IF EXISTS idx_inventory_movements_tenant_id;
DROP TABLE IF EXISTS inventory_movements;

DROP INDEX IF EXISTS idx_inventory_items_sku;
DROP INDEX IF EXISTS idx_inventory_items_tenant_active;
DROP INDEX IF EXISTS idx_inventory_items_tenant_id;
DROP TABLE IF EXISTS inventory_items;

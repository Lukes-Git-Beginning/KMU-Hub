BEGIN;

ALTER TABLE inventory_items DROP COLUMN IF EXISTS location_id;

DROP TABLE IF EXISTS inventory_locations;

COMMIT;

BEGIN;

ALTER TABLE inventory_movements
    DROP COLUMN IF EXISTS notes,
    DROP COLUMN IF EXISTS location_to,
    DROP COLUMN IF EXISTS location_from,
    DROP COLUMN IF EXISTS reference;

ALTER TABLE inventory_items
    DROP COLUMN IF EXISTS linked_purchase_order,
    DROP COLUMN IF EXISTS serial_numbers,
    DROP COLUMN IF EXISTS batch_number,
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS price,
    DROP COLUMN IF EXISTS category;

COMMIT;

-- 000183 Inventar full-extend: neue Felder auf inventory_items und inventory_movements
-- (renumbered von 000170: Prod-DB stand bereits auf 182 nach zeiterfassung-Deploy)

BEGIN;

-- inventory_items: category, price, currency, batch_number, serial_numbers, linked_purchase_order
ALTER TABLE inventory_items
    ADD COLUMN IF NOT EXISTS category             TEXT         NULL,
    ADD COLUMN IF NOT EXISTS price                NUMERIC(12,2) NULL,
    ADD COLUMN IF NOT EXISTS currency             TEXT         NULL DEFAULT 'EUR',
    ADD COLUMN IF NOT EXISTS batch_number         TEXT         NULL,
    ADD COLUMN IF NOT EXISTS serial_numbers       TEXT[]       NULL,
    ADD COLUMN IF NOT EXISTS linked_purchase_order TEXT        NULL;

-- inventory_movements: reference, location_from, location_to, notes
ALTER TABLE inventory_movements
    ADD COLUMN IF NOT EXISTS reference     TEXT NULL,
    ADD COLUMN IF NOT EXISTS location_from TEXT NULL,
    ADD COLUMN IF NOT EXISTS location_to   TEXT NULL,
    ADD COLUMN IF NOT EXISTS notes         TEXT NULL;

COMMIT;

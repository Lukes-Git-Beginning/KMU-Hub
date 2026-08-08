BEGIN;

ALTER TABLE rental_inspections DROP COLUMN checklist;
ALTER TABLE rental_inspections DROP COLUMN signature_data;

COMMIT;

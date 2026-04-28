-- Rollback 000101

ALTER TABLE rental_inspections DROP CONSTRAINT IF EXISTS uq_rental_inspections_kind;
ALTER TABLE rentals DROP CONSTRAINT IF EXISTS uq_rentals_no_overlap;

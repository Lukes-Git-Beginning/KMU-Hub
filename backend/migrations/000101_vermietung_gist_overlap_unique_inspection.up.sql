-- Migration 000101: DB-level rental overlap prevention + unique inspection kind (Bug #4)

-- Require btree_gist extension for GIST exclusion on UUID + tstzrange columns
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Exclude overlapping active rentals for the same object at DB level.
-- Only active rentals (confirmed/active) are checked — cancelled/completed are exempt.
-- tstzrange is half-open: [start, end)
ALTER TABLE rentals
    ADD CONSTRAINT uq_rentals_no_overlap
    EXCLUDE USING GIST (
        tenant_id   WITH =,
        object_id   WITH =,
        tstzrange(start_date, end_date, '[)') WITH &&
    )
    WHERE (status NOT IN ('cancelled', 'completed'));

-- Each rental may have at most one inspection of each kind (handover/return)
ALTER TABLE rental_inspections
    ADD CONSTRAINT uq_rental_inspections_kind
    UNIQUE (tenant_id, rental_id, kind);

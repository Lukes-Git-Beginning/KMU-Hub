-- ============================================================================
-- Migration 000302: rental_inspections.signature_data + rental_inspections.checklist
--
-- Signature persistence already exists for rentals themselves
-- (rentals.signature_data, see the /rentals/{id}/signature endpoint) but not
-- for the handover/return inspection that documents the object's condition.
-- checklist replaces the plan to keep condition notes as pure free text: a
-- structured JSONB array of {label, condition, remark} positions.
--
-- Both columns are nullable: existing rows (and any repository caller that
-- doesn't go through Service.CreateInspection's normalization) have no
-- signature/checklist recorded. The API layer converts a NULL/nil checklist
-- to an empty list when serializing (RentalInspection.checklist), so callers
-- never see JSON null there even though the column itself allows it.
-- ============================================================================

BEGIN;

ALTER TABLE rental_inspections ADD COLUMN signature_data TEXT NULL;
ALTER TABLE rental_inspections ADD COLUMN checklist JSONB NULL;

COMMIT;

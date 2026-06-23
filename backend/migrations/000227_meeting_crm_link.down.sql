-- Rollback Migration 000227

DROP INDEX IF EXISTS meetings_deal_id_idx;
DROP INDEX IF EXISTS meetings_contact_id_idx;

ALTER TABLE meetings DROP COLUMN IF EXISTS deal_id;
ALTER TABLE meetings DROP COLUMN IF EXISTS contact_id;

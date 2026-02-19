-- Migration 000042 DOWN: Remove visibility and owner_id from contacts

DROP INDEX IF EXISTS idx_contacts_owner;
DROP INDEX IF EXISTS idx_contacts_visibility;

ALTER TABLE contacts DROP CONSTRAINT IF EXISTS chk_contacts_visibility;
ALTER TABLE contacts DROP COLUMN IF EXISTS owner_id;
ALTER TABLE contacts DROP COLUMN IF EXISTS visibility;

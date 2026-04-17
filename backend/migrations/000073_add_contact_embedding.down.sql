-- Reverse 000073: drop embedding infrastructure on contacts.
DROP INDEX IF EXISTS idx_contacts_embedding;

ALTER TABLE contacts
    DROP COLUMN IF EXISTS embedding_updated_at,
    DROP COLUMN IF EXISTS embedding,
    DROP COLUMN IF EXISTS search_text;

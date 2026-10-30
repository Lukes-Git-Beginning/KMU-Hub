-- Reverts 000368 to a nullable expiry column. The backfilled expiries stay:
-- which rows had none before cannot be told apart from rows whose author set
-- created_at + 365 days, so the down migration does not guess.
BEGIN;

ALTER TABLE document_share_links ALTER COLUMN expires_at DROP NOT NULL;

COMMIT;

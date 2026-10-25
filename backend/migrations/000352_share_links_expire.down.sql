-- Reverts 000352 to nullable expiry columns. The backfilled expiries stay:
-- which rows had none before cannot be told apart from rows whose author set
-- created_at + 365 days, so the down migration does not guess.
BEGIN;

ALTER TABLE report_share_tokens ALTER COLUMN expires_at DROP NOT NULL;
ALTER TABLE form_share_tokens ALTER COLUMN expires_at DROP NOT NULL;
ALTER TABLE wiki_share_tokens ALTER COLUMN expires_at DROP NOT NULL;

COMMIT;

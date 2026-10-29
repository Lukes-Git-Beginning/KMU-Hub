-- Restores the clear-text `token` column. The clear tokens of links already
-- mailed are gone for good, so those rows get the hash as token only to keep
-- the column unique: their links are dead after this down migration. A row
-- whose hash was minted but whose mail never went out is still usable -- the
-- old dispatcher mails whatever `token` holds. A pending row without a hash
-- yet gets a fresh random token, so it stays pending for the old code.

BEGIN;

ALTER TABLE ticket_csat_responses ADD COLUMN token TEXT NULL;
UPDATE ticket_csat_responses
   SET token = COALESCE(token_hash, replace(gen_random_uuid()::text, '-', '') || replace(gen_random_uuid()::text, '-', ''))
 WHERE token_hash IS NOT NULL OR token_expires_at IS NOT NULL;

DROP INDEX IF EXISTS idx_ticket_csat_responses_due;
DROP INDEX IF EXISTS uq_ticket_csat_responses_token_hash;
ALTER TABLE ticket_csat_responses DROP COLUMN token_hash;

CREATE UNIQUE INDEX uq_ticket_csat_responses_token
    ON ticket_csat_responses (token)
    WHERE token IS NOT NULL;

CREATE INDEX idx_ticket_csat_responses_due
    ON ticket_csat_responses (survey_send_after)
    WHERE token IS NOT NULL
      AND survey_sent_at IS NULL
      AND submitted_at IS NULL;

COMMIT;

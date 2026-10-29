-- Restores the clear-text `token` column. The clear tokens themselves are gone
-- for good, so the column is refilled with the hash only to satisfy NOT NULL
-- UNIQUE: after this down migration every link minted before it is dead, and
-- only links minted afterwards resolve again.

BEGIN;

ALTER TABLE report_share_tokens ADD COLUMN token TEXT;
UPDATE report_share_tokens SET token = token_hash;
ALTER TABLE report_share_tokens
    ALTER COLUMN token SET NOT NULL,
    ADD CONSTRAINT report_share_tokens_token_key UNIQUE (token),
    DROP COLUMN token_hash;

COMMIT;

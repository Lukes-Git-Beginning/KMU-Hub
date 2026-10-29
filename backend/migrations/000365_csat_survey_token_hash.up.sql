-- ============================================================================
-- Migration 000365: CSAT survey links are minted at dispatch, stored as hash.
--
-- 000288 kept the survey token in clear text, minted when the ticket closed
-- and read back by the dispatcher to build the mail. Whoever read the table
-- held working rating links for every pending and sent survey. Decision
-- 2026-10-26 (variant b): the dispatcher mints the token when it claims the
-- survey, stores only its SHA-256 (internal/linksecret) in the same UPDATE and
-- mails the plain token -- it never touches the database.
--
-- A survey is pending while token_expires_at is set: closing a ticket sets the
-- expiry and the send time but no hash; the claim adds the hash; redemption
-- and cancellation clear the expiry. token_hash therefore stays NULL until the
-- mail is about to go out, and a retry after a failed send replaces it.
--
-- Existing links keep working: the backfill hashes the stored token the same
-- way linksecret.Hash does (hex SHA-256 of the UTF-8 bytes, as in 000351).
-- Dropping `token` drops uq_ticket_csat_responses_token and the due index
-- with it; both are recreated on the new columns.
--
-- No RLS change: the tenant_isolation policy is column-independent.
-- ============================================================================

BEGIN;

ALTER TABLE ticket_csat_responses ADD COLUMN token_hash TEXT NULL;
UPDATE ticket_csat_responses
   SET token_hash = encode(sha256(convert_to(token, 'UTF8')), 'hex')
 WHERE token IS NOT NULL;

DROP INDEX IF EXISTS idx_ticket_csat_responses_due;
DROP INDEX IF EXISTS uq_ticket_csat_responses_token;
ALTER TABLE ticket_csat_responses DROP COLUMN token;

-- Global, not per tenant: the public redeem endpoint has no tenant context and
-- resolves the tenant FROM the hash row.
CREATE UNIQUE INDEX uq_ticket_csat_responses_token_hash
    ON ticket_csat_responses (token_hash)
    WHERE token_hash IS NOT NULL;

CREATE INDEX idx_ticket_csat_responses_due
    ON ticket_csat_responses (survey_send_after)
    WHERE token_expires_at IS NOT NULL
      AND survey_sent_at IS NULL
      AND submitted_at IS NULL;

COMMIT;
